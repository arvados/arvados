#!/bin/bash
#
# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0
#
# This script is the entrypoint for a container run by setup_suite.sh to create
# user account fixtures in LDAP.

set -e
set -u
set -o pipefail

result=0
for tries in $(seq 9 -1 0); do
    ldapadd \
        -H ldap://arvados-test-openldap:1389/ \
        -D cn=admin,dc=example,dc=org \
        -w adminpassword \
        -f /add_example_user.ldif ||
        result=$?
    # ldapadd uses exit code 68 to mean "user already exists."
    if [[ "$result" = 0 ]] || [[ "$result" = 68 ]]; then
        exit 0
    elif [[ "$tries" != 0 ]]; then
        sleep 1
    fi
done

echo 'error: failed to add user entry' >&2
exit "$result"
