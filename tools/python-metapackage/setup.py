# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: Apache-2.0

import setuptools
import runpy

from pathlib import Path

arvados_version = runpy.run_path(Path(__file__).with_name('arvados_version.py'))
arv_mod = arvados_version['ARVADOS_PYTHON_MODULES']['arvados-tools']
version = arv_mod.get_version()
setuptools.setup(
    install_requires=[
        *arv_mod.iter_dependencies(version=version, extras={
            'arvados-cluster-activity': ['prometheus'],
        }),
    ],
    version=version,
)
