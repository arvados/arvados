#!/bin/bash -x
#
# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0 OR Apache-2.0
#
set -e

commit_at_dir() {
    git log -n1 --format=%H .
}

build_version() {
    # Output the version being built, or if we're building a
    # dev/prerelease, output a version number based on the git log for
    # the current working directory.
    if [[ -n "$ARVADOS_BUILDING_VERSION" ]]; then
        echo "$ARVADOS_BUILDING_VERSION"
        return
    fi

    $WORKSPACE/build/version-at-commit.sh $(commit_at_dir)
}

exec docker run --rm --user $UID -v $PWD:$PWD -w $PWD gradle:5.3.1 /bin/sh -c 'gradle clean && gradle test && gradle jar install '"-Pversion=$(build_version) $gradle_upload"
