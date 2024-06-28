#!/bin/sh
# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0
set -ex

cd ../../sdk/python
python3 setup.py sdist
SDK_VERSION=$(python3 arvados_version.py)

cd ../../tools/crunchstat-summary
python3 setup.py sdist
CRUNCHSTAT_VERSION=$(python3 arvados_version.py)

cd ../../tools/cluster-activity
python3 setup.py sdist
VERSION=$(python3 arvados_version.py)

rm -f docker/*.tar.gz
cp ../../sdk/python/dist/arvados-python-client-${SDK_VERSION}.tar.gz \
   ../crunchstat-summary/dist/crunchstat_summary-${CRUNCHSTAT_VERSION}.tar.gz \
   dist/arvados-user-activity-${VERSION}.tar.gz \
   docker/
cd docker
docker build -t arvados/cluster-activity:$VERSION .
