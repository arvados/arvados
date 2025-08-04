# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

import setuptools
import runpy

from pathlib import Path

arvados_version = runpy.run_path(Path(__file__).with_name('arvados_version.py'))
arv_mod = arvados_version['ARVADOS_PYTHON_MODULES']['arvados_fuse']
version = arv_mod.get_version()
setuptools.setup(
    cmdclass=arvados_version['CMDCLASS'],
    install_requires=[
        *arv_mod.iter_dependencies(version=version),
        'arvados-llfuse >= 1.5.1',
        'python-daemon',
        'ciso8601 >= 2.0.0',
        'setuptools',
        "prometheus_client"
    ],
    version=version,
)
