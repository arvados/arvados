#!/bin/bash -x
#
# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0 OR Apache-2.0
#
set -e

format_last_commit_here() {
    local format="$1"; shift
    TZ=UTC git log -n1 --first-parent "--format=format:$format" .
}

version_from_git() {
    # Output the version being built, or if we're building a
    # dev/prerelease, output a version number based on the git log for
    # the current working directory.
    if [[ -n "$ARVADOS_BUILDING_VERSION" ]]; then
        echo "$ARVADOS_BUILDING_VERSION"
        return
    fi

    local git_ts git_hash prefix
    if [[ -n "$1" ]] ; then
        prefix="$1"
    else
        prefix="0.1"
    fi

    declare $(format_last_commit_here "git_ts=%ct git_hash=%h")
    ARVADOS_BUILDING_VERSION="$(git describe --abbrev=0).$(date -ud "@$git_ts" +%Y%m%d%H%M%S)"
    echo "$ARVADOS_BUILDING_VERSION"
}

nohash_version_from_git() {
    version_from_git $1 | cut -d. -f1-3
}

timestamp_from_git() {
    format_last_commit_here "%ct"
}
if [[ -n "$1" ]]; then
    build_version="$1"
else
    build_version="$(version_from_git)"
fi
#UID=$(id -u) # UID is read-only on many systems
exec docker run --rm --user $UID -v $PWD:$PWD -w $PWD gradle:5.3.1 /bin/sh -c 'gradle clean && gradle test && gradle jar install '"$gradle_upload"
