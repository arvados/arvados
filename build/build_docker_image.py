#!/usr/bin/env python3
# build_docker_image.py - Build a Docker image with Python source packages
#
# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0
#
# Requires you have requirements.build.txt installed

import argparse
import logging
import os
import runpy
import shlex
import shutil
import subprocess
import sys
import tempfile

from pathlib import Path

logger = logging.getLogger('build_docker_image')
_null_loghandler = logging.NullHandler()
logger.addHandler(_null_loghandler)

def _log_cmd(level, msg, *args):
    *args, cmd = args
    if logger.isEnabledFor(level):
        logger.log(level, f'{msg}: %s', *args, ' '.join(shlex.quote(s) for s in cmd))


def _log_and_run(cmd, *, level=logging.DEBUG, check=True, **kwargs):
    _log_cmd(level, "running command", cmd)
    return subprocess.run(cmd, check=check, **kwargs)


class OptionError(ValueError):
    pass


class DockerImage:
    _BUILD_ARGS = {}
    _REGISTRY = {}

    @classmethod
    def register(cls, subcls):
        cls._REGISTRY[subcls.NAME] = subcls
        pre_name, _, shortname = subcls.NAME.rpartition('/')
        if pre_name == 'arvados':
            cls._REGISTRY[shortname] = subcls
        return subcls

    @classmethod
    def build_from_args(cls, args):
        try:
            subcls = cls._REGISTRY[args.docker_image]
        except KeyError:
            raise OptionError(f"unrecognized Docker image {args.docker_image!r}") from None
        else:
            return subcls(args)

    def __init__(self, args):
        self.extra_args = args.extra_args
        self.workspace = args.workspace
        if args.tag is not None:
            self.tag = args.tag
        elif version := (args.version or self.dev_version()):
            self.tag = f'{self.NAME}:{version}'
        else:
            self.tag = None

    def __enter__(self):
        tmpname = self.NAME.replace('/', '-')
        self.context_dir = Path(tempfile.mkdtemp(prefix=f'{tmpname}.'))
        return self

    def __exit__(self, exc_type, exc_value, exc_tb):
        shutil.rmtree(self.context_dir, ignore_errors=True)
        del self.context_dir

    def build_docker_image(self):
        logger.info("building Docker image %s", self.tag or self.NAME)
        cmd = ['docker', 'image', 'build']
        cmd.extend(
            f'--build-arg={key}={val}'
            for key, val in self._BUILD_ARGS.items()
        )
        cmd.append(f'--file={self.workspace / self.DOCKERFILE_PATH}')
        if self.tag is not None:
            cmd.append(f'--tag={self.tag}')
        cmd.append(str(self.context_dir))
        return _log_and_run(cmd)

    def dev_version(self):
        return None


class PythonVenvImage(DockerImage):
    DOCKERFILE_PATH = 'build/docker/python-venv.Dockerfile'
    _EXTRAS = {}
    _TEST_COMMAND = None

    def __init__(self, args):
        arv_vars = runpy.run_path(args.workspace / 'sdk/python/arvados_version.py')
        self.arv_pymod = arv_vars['ARVADOS_PYTHON_MODULES'][self._PACKAGE_NAME]
        super().__init__(args)

    def dev_version(self):
        return self.arv_pymod.get_version(self.workspace / self.arv_pymod.src_path)

    def build_python_wheel(self, src_dir):
        logger.info("building Python wheel at %s", src_dir)
        cmd = [sys.executable, '-m', 'build', '--outdir', str(self.context_dir)]
        return _log_and_run(cmd, cwd=src_dir, umask=0o022)

    def build_requirements(self):
        with (self.context_dir / 'requirements.txt').open('w') as requirements_file:
            for whl_path in self.context_dir.glob('*.whl'):
                name, _, _ = whl_path.stem.partition('-')
                try:
                    name += f' [{self._EXTRAS[name]}]'
                except KeyError:
                    pass
                whl_uri = Path('/usr/local/src', whl_path.name).as_uri()
                print(name, '@', whl_uri, file=requirements_file)

    def build_docker_image(self):
        for path in self.extra_args:
            self.build_python_wheel(path)
        for dep in self.arv_pymod.dependencies:
            self.build_python_wheel(self.workspace / dep.src_path)
        self.build_python_wheel(self.workspace / self.arv_pymod.src_path)
        self.build_requirements()
        result = super().build_docker_image()
        if self.tag and self._TEST_COMMAND:
            _log_and_run(
                ['docker', 'run', '--rm', '--tty', self.tag] + self._TEST_COMMAND,
                stdout=subprocess.DEVNULL,
            )
        return result


@DockerImage.register
class ClusterActivityImage(PythonVenvImage):
    NAME = 'arvados/cluster-activity'
    _BUILD_ARGS = {
        'APT_PKGLIST': 'libcurl4',
        'OLD_PKGNAME': 'python3-arvados-cluster-activity',
    }
    _EXTRAS = {
        'arvados_cluster_activity': 'prometheus',
    }
    _PACKAGE_NAME = 'arvados-cluster-activity'
    _TEST_COMMAND = ['arv-cluster-activity', '--version']


@DockerImage.register
class JobsImage(PythonVenvImage):
    NAME = 'arvados/jobs'
    _BUILD_ARGS = {
        'APT_PKGLIST': 'libcurl4 nodejs',
        'OLD_PKGNAME': 'python3-arvados-cwl-runner',
    }
    _PACKAGE_NAME = 'arvados-cwl-runner'
    _TEST_COMMAND = ['arvados-cwl-runner', '--version']


class Environments:
    @staticmethod
    def production(args):
        if args.version is None:
            raise OptionError(
                "$ARVADOS_BUILDING_VERSION must be set to build production images"
            )

    @staticmethod
    def development(args):
        return

    _ARG_MAP = {
        'dev': development,
        'devel': development,
        'development': development,
        'prod': production,
        'production': production,
    }

    @classmethod
    def parse_argument(cls, s):
        try:
            return cls._ARG_MAP[s.lower()]
        except KeyError:
            raise ValueError(f"unrecognized environment {s!r}")


class UploadActions:
    @staticmethod
    def to_arvados(tag):
        logger.info("uploading Docker image %s to Arvados", tag)
        name, _, version = tag.rpartition(':')
        if name:
            cmd = ['arv-keepdocker', name, version]
        else:
            cmd = ['arv-keepdocker', tag]
        return _log_and_run(cmd)

    @staticmethod
    def to_docker_hub(tag):
        logger.info("uploading Docker image %s to Docker Hub", tag)
        cmd = ['docker', 'push', tag]
        for tries_left in range(4, -1, -1):
            try:
                docker_push = _log_and_run(cmd)
            except subprocess.CalledProcessError:
                if tries_left == 0:
                    raise
            else:
                break
        return docker_push

    _ARG_MAP = {
        'arv-keepdocker': to_arvados,
        'arvados': to_arvados,
        'docker': to_docker_hub,
        'docker_hub': to_docker_hub,
        'dockerhub': to_docker_hub,
        'keepdocker': to_arvados,
    }

    @classmethod
    def parse_argument(cls, s):
        try:
            return cls._ARG_MAP[s.lower()]
        except KeyError:
            raise ValueError(f"unrecognized upload method {s!r}")


class ArgumentParser(argparse.ArgumentParser):
    def __init__(self):
        super().__init__(
            prog='build_docker_image.py',
            usage='%(prog)s [options ...] IMAGE_NAME [source directory ...]',
        )
        # We put environment variables for the tool in the args so the rest
        # of the program has a single place to access parameters.
        env_workspace = os.environ.get('WORKSPACE')
        self.set_defaults(
            version=os.environ.get('ARVADOS_BUILDING_VERSION'),
            workspace=Path(env_workspace) if env_workspace else None,
        )

        self.add_argument(
            '--environment',
            type=Environments.parse_argument,
            default=Environments.production,
            help="""One of `development` or `production`.
Your build settings will use defaults and be validated based on this setting.
Default is `production` because it's the strictest.
""")

        self.add_argument(
            '--loglevel',
            type=self._parse_loglevel,
            default=logging.WARNING,
            help="""Log level to use, like `debug`, `info`, `warning`, or `error`
""")

        self.add_argument(
            '--tag', '-t',
            help="""Tag for the built Docker image.
Default is generated from the image name and build version.
""")

        self.add_argument(
            '--upload-to',
            type=UploadActions.parse_argument,
            help="""After successfully building the Docker image, upload it to
this destination. Choices are `arvados` or `docker_hub`. Both require
credentials in place to work.
""")

        self.add_argument(
            'docker_image',
            metavar='IMAGE_NAME',
            choices=sorted(DockerImage._REGISTRY),
            help="""Docker image to build.
Supported images are: %(choices)s.
""")

        self.add_argument(
            'extra_args',
            metavar='SOURCE_DIR',
            type=Path,
            nargs=argparse.ZERO_OR_MORE,
            default=[],
            help="""Before building the Docker image, the tool will build a
Python wheel from each source directory and add it to the Docker build context.
You can use this during testing to install specific development versions of
dependencies.
""")

    def _parse_loglevel(self, s):
        try:
            return logging.getLevelNamesMapping()[s.upper()]
        except KeyError:
            raise ValueError(f"unrecognized logging level {s!r}")


def main(args):
    if not isinstance(args, argparse.Namespace):
        args = ArgumentParser().parse_args(args)
    if args.workspace is None:
        raise OptionError("$WORKSPACE must be set to the Arvados source directory")
    args.environment(args)
    docker_image = DockerImage.build_from_args(args)
    if args.upload_to and not docker_image.tag:
        raise OptionError("cannot upload a Docker image without a tag")
    with docker_image:
        docker_image.build_docker_image()
    if args.upload_to:
        args.upload_to(docker_image.tag)
    return os.EX_OK


if __name__ == '__main__':
    argparser = ArgumentParser()
    _args = argparser.parse_args()
    logging.basicConfig(
        format=f'{logger.name}: %(levelname)s: %(message)s',
        level=_args.loglevel,
    )
    try:
        returncode = main(_args)
    except OptionError as err:
        argparser.error(err.args[0])
        returncode = 2
    except subprocess.CalledProcessError as err:
        _log_cmd(
            logging.ERROR,
            "command failed with exit code %s",
            err.returncode,
            err.cmd,
        )
        returncode = err.returncode
    exit(returncode)
