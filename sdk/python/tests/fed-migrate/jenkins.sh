#!/bin/bash
# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: Apache-2.0

if test -z "$WORKSPACE" ; then
    echo "WORKSPACE unset"
    exit 1
fi

docker stop fedbox1 fedbox2 fedbox3
docker rm fedbox1 fedbox2 fedbox3
docker rm fedbox1-data fedbox2-data fedbox3-data

set -ex

mkdir -p "$WORKSPACE/tmp/arvbox"
python3 -m venv "$WORKSPACE/tmp/venv3"
"$WORKSPACE/tmp/venv3/bin/pip" install -e "$WORKSPACE/sdk/python" "$WORKSPACE/sdk/cwl"
alias cwltool='"$WORKSPACE/tmp/venv3/bin/cwltool"'

export PATH=$PATH:$WORKSPACE/tools/arvbox/bin

cd $WORKSPACE/sdk/python/tests/fed-migrate
cwltool arvbox-make-federation.cwl \
	--arvbox_base $WORKSPACE/tmp/arvbox \
	--branch $(git rev-parse HEAD) \
	--arvbox_mode localdemo > fed.json

cwltool fed-migrate.cwl fed.json
