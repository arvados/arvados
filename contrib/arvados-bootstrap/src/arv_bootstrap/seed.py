# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: Apache-2.0

import argparse
import contextlib
import dataclasses
import functools
import json
import logging
import logging.handlers
import pathlib
import os
import re
import sys
import traceback
import urllib.parse

from collections import abc

import arvados
import arvados.commands._util as cmd_util

logger = logging.getLogger('arvados.commands.seed')
_root_logger = logging.getLogger()

def is_mapping(arg):
    return isinstance(arg, abc.Mapping)


@dataclasses.dataclass
class ExceptHook:
    logger: logging.Logger
    exit_code: int = os.EX_SOFTWARE

    def __call__(self, exc_type, exc_value, exc_tb):
        self.logger.critical(
            "internal %s: %s", exc_type.__name__, exc_value,
            exc_info=self.logger.isEnabledFor(logging.DEBUG),
        )
        raise SystemExit(self.exit_code)

    @contextlib.contextmanager
    def using_exit_code(self, exit_code):
        orig_code = self.exit_code
        self.exit_code = exit_code
        # We intentionally *don't* want to use `finally` here, because we don't
        # want to restore the original code for an unhandled exception.
        yield self
        self.exit_code = orig_code


class Path(pathlib.Path):
    _flavour = pathlib._posix_flavour

    def __format__(self, format_spec=''):
        if format_spec.endswith('`'):
            return f"`{super().__format__(format_spec[:-1])}`"
        else:
            return super().__format__(format_spec)


@dataclasses.dataclass
class ArvadosResources:
    _resources: abc.Mapping

    @classmethod
    def from_client(cls, arv_client):
        return cls(arv_client._resourceDesc['resources'])

    @staticmethod
    def _sep_caps(match):
        return '_'.join(match.group(0).lower())

    def singular_name(self, name):
        if name == 'sys':
            return name
        elif name.endswith('ies'):
            return f'{name[:-3]}y'
        else:
            return name.removesuffix('s')

    def canonical_name(self, name):
        s = re.sub(r'\W', '_', name)
        s = re.sub(r'[a-z][A-Z]', self._sep_caps, s)
        s = s.lower()
        if s in self._resources:
            return s
        elif s.endswith('y'):
            s = f'{s[:1]}ies'
        else:
            s += 's'
        if s in self._resources:
            return s
        raise ValueError(f"no resource found for {name!r}")

    def parameters(self, resource_name, method_name):
        return self._resources[resource_name]['methods'][method_name]['parameters']


@dataclasses.dataclass
class DirectoryLoader:
    arv_client: arvados.api.ThreadSafeAPIClient
    base: abc.Mapping | None
    params: abc.Mapping | None
    resources: ArvadosResources

    OBJECT_BASE_PATH = Path('arvados_seed_object.json')
    PARAMETERS_PATH = Path('arvados_seed_parameters.json')

    @classmethod
    def from_args(cls, args):
        arv_client = arvados.api.api(**args.api_kwargs)
        return cls(
            arv_client,
            args.object_base,
            args.parameters,
            ArvadosResources.from_client(arv_client),
        )

    def _load_defaults(self, instance_defaults, path):
        if instance_defaults is not None:
            return instance_defaults
        try:
            with path.open('rb') as json_file:
                defaults = json.load(json_file)
        except FileNotFoundError:
            defaults = {}
        if not is_mapping(defaults):
            raise ValueError(f"{path:`} does not contain a JSON object")
        return defaults

    def _create_from(self, json_path, base, base_params):
        prefix, _, rname = json_path.stem.rpartition('.')
        if not prefix:
            raise ValueError(f"{path:`} does have an object type in the name")
        rname = self.resources.canonical_name(rname)

        kwargs = dict(base_params)
        if 'ensure_unique_name' in self.resources.parameters(rname, 'create'):
            kwargs.setdefault('ensure_unique_name', True)

        with json_path.open('rb') as json_file:
            json_body = json.load(json_file)
        kwargs['body'] = {self.resources.singular_name(rname): base | json_body}

        resource = getattr(self.arv_client, rname)
        return resource().create(**kwargs).execute()

    def build_from(self, dir_path):
        base = self._load_defaults(self.base, dir_path / self.OBJECT_BASE_PATH)
        base_params = self._load_defaults(self.params, dir_path / self.PARAMETERS_PATH)
        created = {}
        failed = {}
        for path in sorted(dir_path.glob('*.json')):
            path_key = str(path.absolute())
            try:
                result = self._create_from(path, base, base_params)
            except Exception as err:
                logger.warning(
                    "failed to load %s: %s", path, err,
                    exc_info=logger.isEnabledFor(logging.DEBUG),
                )
                failed[path_key] = str(err)
            else:
                created[path_key] = result
        return (created, failed)


class ConfigLoader:
    DEFAULT_CONFIG_PATH = Path('/etc/arvados/config.yml')
    DISCOVERY_SERVICE_PATH = 'discovery/v1/apis/{api}/{apiVersion}/rest'

    @classmethod
    def _load_yaml(cls, path):
        try:
            with open(path, 'rb') as yaml_file:
                result = yaml.safe_load(yaml_file)
        except OSError as err:
            raise ValueError(f"error reading {path:`}: {err}") from None
        if not is_mapping(result):
            raise ValueError(f"{path:`} is not a YAML object")
        return result

    @classmethod
    def _cluster_config_path(cls):
        return Path(os.environ.get('ARVADOS_CONFIG', cls.DEFAULT_CONFIG_PATH))

    @classmethod
    def _from_one_cluster(cls, config):
        try:
            controller_url = config['Services']['Controller']['ExternalURL']
            token = config['SystemRootToken']
        except (KeyError, TypeError) as err:
            raise ValueError(f"error loading cluster configuration: {err}") from None
        try:
            insecure = config['TLS']['Insecure']
        except (KeyError, TypeError):
            insecure = False
        return {
            'version': 'v1',
            'discoveryServiceUrl': urllib.parse.urljoin(controller_url, cls.DISCOVERY_SERVICE_PATH),
            'token': token,
            'insecure': insecure,
        }

    @classmethod
    def from_cluster(cls, arg):
        path = cls._cluster_config_path()
        whole_config = cls._load_yaml(path)
        try:
            configs = whole_config['Clusters'].items()
        except (AttributeError, KeyError, TypeError) as err:
            raise ValueError(f"error loading clusters configuration: {err}") from None
        kwargs = None
        kwargs_id = None
        for cluster_id, config in configs:
            try:
                new_kwargs = cls._from_one_cluster(config)
            except ValueError:
                continue
            if kwargs is None:
                kwargs = new_kwargs
                kwargs_id = cluster_id
            else:
                raise ValueError(
                    f"{path:`} has configuration for both {kwargs_id} and {cluster_id} - "
                    "specify a cluster ID",
                )
        if kwargs is None:
            raise ValueError(f"no usable cluster configuration found in {path:`}")
        else:
            return kwargs

    @classmethod
    def from_cluster_id(cls, arg):
        path = cls._cluster_config_path()
        whole_config = cls._load_yaml(path)
        try:
            config = whole_config['Clusters'][arg]
        except (AttributeError, KeyError, TypeError) as err:
            raise ValueError(f"error loading {arg} configuration from {path:`}: {err}") from None
        return self._from_one_cluster(config)

    @classmethod
    def from_env(cls, arg):
        arvados.config.initialize('')
        return cls.from_user()

    @classmethod
    def from_user(cls, arg):
        return arvados.api.api_kwargs_from_config('v1')

    @classmethod
    def parse_arg(cls, arg):
        try:
            constructor = getattr(cls, f'from_{arg}')
        except AttributeError:
            if re.fullmatch(r'^[a-z0-9]$', arg):
                constructor = cls.from_cluster_id
            else:
                raise ValueError(f"invalid configuration source {arg!r}") from None
        return constructor(arg)

    @classmethod
    def default_config(cls):
        if os.geteuid() == 0:
            return cls.from_cluster(None)
        else:
            return cls.from_user(None)


def parse_loglevel(arg):
    try:
        return logging.getLevelNamesMapping()[arg.upper()]
    except KeyError:
        raise ValueError(f"invalid log level {arg!r}") from None


def validate_mapping(arg):
    if is_mapping(arg):
        return arg
    else:
        raise ValueError("value is not a JSON object")


def parse_arguments(arglist=None):
    parser = argparse.ArgumentParser(
        prog="arv-seed",
        description="Create multiple Arvados objects from a directory of JSON files",
    )
    parser.add_argument(
        '--client-from',
        metavar='SOURCE',
        type=ConfigLoader.parse_arg,
        dest='api_kwargs',
        help="""
Where to find the Arvados API server and token. Specify one of a cluster ID,
`cluster`, `env`, or `user`. The first two options load the cluster configuration
file from `$ARVADOS_CONFIG` or `/etc/arvados/config.yml`.
""")
    parser.add_argument(
        '--loglevel',
        metavar='LEVEL',
        type=parse_loglevel,
        default=logging.INFO,
        help="""
The name of a log level like `debug`, `info`, `warning`, or `error`
""")
    parser.add_argument(
        '--object-base', '--base',
        metavar='BASE_JSON',
        type=cmd_util.JSONArgument(validate_mapping),
        help="""
JSON object or path to set common attributes for all created objects.
If not set, will try to read `arvados_seed_object.json` in each directory.
""")
    parser.add_argument(
        '--parameters', '--params',
        metavar='PARAMS_JSON',
        type=cmd_util.JSONArgument(validate_mapping),
        help="""
JSON object or path to set parameters when creating objects.
If not set, will try to read `arvados_seed_parameters.json` in each directory.
""")
    parser.add_argument(
        'dir_paths',
        metavar='DIRECTORY',
        type=Path,
        nargs=argparse.ONE_OR_MORE,
        help="""
Directory to read object JSON files from. Object files must be named
`<name>.<type>.json`, where `type` is an Arvados API resource type.
""")
    args = parser.parse_args(arglist)
    if args.api_kwargs is None:
        args.api_kwargs = ConfigLoader.default_config()
    return args


def add_log_handlers(logger, stderr=sys.stderr):
    syslog = logging.handlers.SysLogHandler('/dev/log')
    syslog.setFormatter(logging.Formatter('[%(name)s] %(message)s'))
    logger.addHandler(syslog)
    if os.environ.get('TERM'):
        stream = logging.StreamHandler(stderr)
        stream.setFormatter(logging.Formatter(
            '[%(asctime)s] arv-seed: %(levelname)s: %(message)s',
            '%Y-%m-%d %H:%M:%S',
        ))
        logger.addHandler(stream)


def main(
        arglist=None,
        *,
        stdout=sys.stdout,
        stderr=sys.stderr,
        is_main=Path(sys.argv[0]).stem == 'arv-seed',
):
    if is_main:
        add_log_handlers(_root_logger)
        sys.excepthook = ExceptHook(logger)
        arvados.logger.removeHandler(arvados.logging.log_handler)
    args = parse_arguments(arglist)
    if is_main:
        _root_logger.setLevel(args.loglevel)
        setup_ctx = sys.excepthook.using_exit_code(os.EX_CONFIG)
    else:
        logger.setLevel(args.loglevel)
        setup_ctx = contextlib.nullcontext()
    with setup_ctx:
        loader = DirectoryLoader.from_args(args)

    created = {}
    failed = {}
    for dir_path in args.dir_paths:
        try:
            dir_created, dir_failed = loader.build_from(dir_path)
        except Exception as err:
            logger.warning(
                "failed to load directory %s: %s", dir_path, err,
                exc_info=logger.isEnabledFor(logging.DEBUG),
            )
            failed[dir_path] = err
        else:
            created.update(dir_created)
            failed.update(dir_failed)
    json.dump({'created': created, 'failed': failed}, stdout)
    print(file=stdout)

    if created and failed:
        return 12
    elif failed:
        return 11
    elif created:
        return os.EX_OK
    else:
        return os.EX_NOINPUT
