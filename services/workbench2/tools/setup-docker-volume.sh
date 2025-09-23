#!/bin/bash
# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0
#
# This script runs inside an arvados/workbench Docker container to
# initialize the home volume for test runs.

set -euo pipefail
DEV_USER="$(id --user --name 1000)"
DEV_HOME="/mnt/$DEV_USER"

run_setup() {
    local rundir="$1"; shift
    sudo -u "$DEV_USER" env -C "$rundir" HOME="$DEV_HOME" "$@"
}

cd "/home/$DEV_USER"
# The virtualenv needs to use the /home path, and will write everything there
# regardless of other settings, so we must set it up fully before copying the
# home directory.
run_setup . python3 -m venv VENV3DIR
while read key val; do
    run_setup VENV3DIR bin/pip config --site set "$key" "$val"
done <<EOF
global.disable-pip-version-check true
global.no-cache-dir true
global.no-input true
global.no-python-version-warning true
install.progress-bar off
EOF
grep --no-filename -E '^yq[^-_[:alnum:]]' "$ARVADOS_DIRECTORY"/build/requirements.*.txt |
    run_setup VENV3DIR xargs -d\\n bin/pip install
cp --archive . "$DEV_HOME"

# Now install everything else directly to the volume.
cd "$ARVADOS_DIRECTORY"
run_setup cmd/arvados-server go install
run_setup services/workbench2 yarn run cypress install
run_setup services/workbench2 yarn run cypress verify
