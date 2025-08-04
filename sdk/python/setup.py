# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: Apache-2.0

import setuptools
import runpy

from pathlib import Path

arvados_version = runpy.run_path(Path(__file__).with_name('arvados_version.py'))

class BuildDiscoveryPydoc(setuptools.Command):
    """Generate Arvados API documentation

    This class implements a setuptools subcommand, so it follows
    [the SubCommand protocol][1]. Most of these methods are required by that
    protocol, except `should_run`, which we register as the subcommand
    predicate.

    [1]: https://setuptools.pypa.io/en/latest/userguide/extension.html#setuptools.command.build.SubCommand
    """
    user_options = [
        ('discovery-json=', 'J', 'JSON discovery document used to build pydoc'),
        ('discovery-output=', 'O', 'relative path to write discovery document pydoc'),
    ]

    def initialize_options(self):
        self.build_lib = None
        self.discovery_json = 'arvados-v1-discovery.json'
        self.discovery_output = str(Path('arvados', 'api_resources.py'))

    def _relative_path(self, src, optname):
        retval = Path(src)
        if retval.is_absolute():
            raise Exception(f"--{optname} should be a relative path")
        else:
            return retval

    def finalize_options(self):
        self.set_undefined_options("build_py", ("build_lib", "build_lib"))
        self.json_path = self._relative_path(self.discovery_json, 'discovery-json')
        self.out_path = Path(
            self.build_lib,
            self._relative_path(self.discovery_output, 'discovery-output'),
        )

    def run(self):
        discovery2pydoc = runpy.run_path(Path(__file__).with_name('discovery2pydoc.py'))
        arglist = ['--output-file', str(self.out_path), str(self.json_path)]
        returncode = discovery2pydoc['main'](arglist)
        if returncode != 0:
            raise Exception(f"discovery2pydoc exited {returncode}")

    def get_outputs(self):
        return [str(self.out_path)]

    def get_source_files(self):
        return [self.discovery_json]

    def get_output_mapping(self):
        return {
            str(self.out_path): self.discovery_json,
        }


class ArvadosBuild(arvados_version['ArvadosBuildCommand']):
    sub_commands = [
        *arvados_version['ArvadosBuildCommand'].sub_commands,
        ('build_discovery_pydoc', None),
    ]


arv_mod = arvados_version['ARVADOS_PYTHON_MODULES']['arvados-python-client']
version = arv_mod.get_version()
setuptools.setup(
    version=version,
    cmdclass={
        'build': ArvadosBuild,
        'build_arvados_version': arvados_version['BuildArvadosVersion'],
        'build_discovery_pydoc': BuildDiscoveryPydoc,
    },
    install_requires=[
        *arv_mod.iter_dependencies(version=version),
        'boto3',
        'ciso8601 >= 2.0.0',
        'google-api-python-client >= 2.1.0',
        'google-auth',
        'httplib2 >= 0.9.2',
        'pycurl >= 7.19.5.1',
        'websockets >= 11.0',
    ],
)
