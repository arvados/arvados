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
        self.tag = args.tag
        self.workspace = args.workspace

    def __enter__(self):
        tmpname = self.NAME.replace('/', '-')
        self.context_dir = Path(tempfile.mkdtemp(prefix=f'{tmpname}.'))
        return self

    def __exit__(self, exc_type, exc_value, exc_tb):
        shutil.rmtree(self.context_dir, ignore_errors=True)
        del self.context_dir

    def _arvados_version(self, src_dir):
        ver_mod = runpy.run_path(self.workspace / src_dir / 'arvados_version.py')
        return ver_mod['get_version']()

    def build_python_wheel(self, src_dir):
        logger.info("building Python wheel at %s", src_dir)
        if (src_dir / 'pyproject.toml').exists():
            cmd = [sys.executable, '-m', 'build',
                   '--outdir', str(self.context_dir)]
        else:
            cmd = [sys.executable, 'setup.py', 'bdist_wheel',
                   '--dist-dir', str(self.context_dir)]
        return _log_and_run(cmd, cwd=src_dir, umask=0o022)

    def build_docker_image(self):
        logger.info("building Docker image %s", self.tag or self.NAME)
        cmd = ['docker', 'image', 'build']
        cmd.append(f'--file={self.workspace / self.DOCKERFILE_PATH}')
        if self.tag is not None:
            cmd.append(f'--tag={self.tag}')
        cmd.append(str(self.context_dir))
        return _log_and_run(cmd)


@DockerImage.register
class DevJobsImage(DockerImage):
    DOCKERFILE_PATH = 'build/docker/dev-jobs.Dockerfile'
    NAME = 'arvados/dev-jobs'

    def __init__(self, args):
        super().__init__(args)
        if self.tag is None:
            version = args.version or self._arvados_version('sdk/cwl')
            self.tag = f'{self.NAME}:{version}'

    def build_docker_image(self):
        self.build_python_wheel(self.workspace / 'sdk/python')
        self.build_python_wheel(self.workspace / 'tools/crunchstat-summary')
        self.build_python_wheel(self.workspace / 'sdk/cwl')
        super().build_docker_image()
        if self.tag:
            # Check the build was successful.
            _log_and_run([
                'docker', 'run', '--rm', '--tty', self.tag,
                 'arvados-cwl-runner', '--version',
            ], stdout=subprocess.DEVNULL)


@DockerImage.register
class JobsImage(DevJobsImage):
    NAME = 'arvados/jobs'

    def __init__(self, args):
        if args.tag is None and args.version is None:
            raise OptionError(f"$ARVADOS_BUILDING_VERSION must be set to build {self.NAME}")
        super().__init__(args)


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
            'source_paths',
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
    docker_image = DockerImage.build_from_args(args)
    if args.upload_to and not docker_image.tag:
        raise OptionError("cannot upload a Docker image without a tag")
    with docker_image:
        for path in args.source_paths:
            docker_image.build_python_wheel(path)
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
