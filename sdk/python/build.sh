#!/bin/sh
#
# Apparently the only reliable way to distribute Python packages with pypi and
# install them via pip is as source packages (sdist).
#
# That means that setup.py is run on the system the package is being installed on,
# outside of the Arvados git tree.
#
# In turn, this means that we can not build the minor_version on the fly when
# setup.py is being executed. Instead, we use this script to generate a 'static'
# version of setup.py which will can be distributed via pypi.

minor_version=`git log --format=format:%ct.%h -n1 .`

sed "s|%%MINOR_VERSION%%|$minor_version|" < setup.py.src > setup.py

