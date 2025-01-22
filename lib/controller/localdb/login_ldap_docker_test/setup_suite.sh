#!/bin/bash
#
# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0
#
# This script determines and records what image should be used for test
# controller containers, then starts an LDAP server and adds user account
# fixtures to it.

set -e
set -u
set -o pipefail

cd "$(dirname "$0")"
net_name="$1"; shift
tmpdir="$1"; shift

. /etc/os-release
case "$ID" in
    debian|ubuntu)
        controller_image="$ID:$VERSION_ID"
        ;;
    *)
        echo >&2 "don't know what Docker image corresponds to $NAME $VERSION"
        exit 3  # EXIT_NOTIMPLEMENTED
        ;;
esac
# Pull the image if we don't have it already
docker run --rm "$controller_image" true
echo "$controller_image" >"$tmpdir/controller_image"

go build -o "${tmpdir}" ../../../../cmd/arvados-server

docker run --rm --detach \
       --name=arvados-test-openldap \
       --network="$net_name" \
       osixia/openldap:1.3.0

awk -v passhash="$(docker exec -i arvados-test-openldap slappasswd -s "secret")" -- '
($1 == "userPassword:") { $2 = passhash; }
{ print; }
' add_example_user.ldif >"$tmpdir/add_example_user.ldif"

docker run --rm \
       --entrypoint=/setup_suite_users.sh \
       --network="$net_name" \
       -v "$PWD/setup_suite_users.sh":/setup_suite_users.sh:ro \
       -v "${tmpdir}/add_example_user.ldif":/add_example_user.ldif:ro \
       osixia/openldap:1.3.0
