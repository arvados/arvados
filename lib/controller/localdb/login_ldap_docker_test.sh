#!/bin/bash

# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

# This script demonstrates using LDAP for Arvados user authentication.
#
# It configures arvados controller in a docker container, optionally
# with pam_ldap(5) configured to authenticate against an OpenLDAP
# server in a second docker container.
#
# After adding a "foo" user entry, it uses curl to check that the
# Arvados controller's login endpoint accepts the "foo" account
# username/password and rejects invalid credentials.
#
# It is intended to be run inside .../build/run-tests.sh (in
# interactive mode: "test lib/controller/localdb -tags=docker
# -check.f=LDAP -check.vv"). It assumes ARVADOS_TEST_API_HOST points
# to a RailsAPI server and the desired version of arvados-server is
# installed in $GOPATH/bin.

set -e -o pipefail

debug=/dev/null
if [[ -n ${ARVADOS_DEBUG} ]]; then
    debug=/dev/stderr
    set -x
fi

case "${config_method}" in
    pam | ldap)
        ;;
    *)
        echo >&2 "\$config_method env var must be 'pam' or 'ldap'"
        exit 1
        ;;
esac

hostname="$(hostname)"
tmpdir="$(mktemp -d)"
cleanup() {
    trap - ERR
    rm -r ${tmpdir}
    for h in ${ldapctr} ${ctrlctr}; do
        if [[ -n ${h} ]]; then
            docker kill ${h}
        fi
    done
}
trap cleanup ERR

if [[ -z "$(docker image ls -q osixia/openldap:1.3.0)" ]]; then
    echo >&2 "Pulling docker image for ldap server"
    docker pull osixia/openldap:1.3.0
fi

ldapctr=ldap-${RANDOM}
echo >&2 "Starting ldap server in docker container ${ldapctr}"
docker run --rm --detach \
       -p 389 -p 636 \
       --name=${ldapctr} \
       osixia/openldap:1.3.0
docker logs --follow ${ldapctr} 2>$debug >$debug &
ldaphostports=$(docker port ${ldapctr} 389/tcp)
ldapport=${ldaphostports##*:}
ldapurl="ldap://${hostname}:${ldapport}"
passwordhash="$(docker exec -i ${ldapctr} slappasswd -s "secret")"

# These are the default admin credentials for osixia/openldap:1.3.0
adminuser=admin
adminpassword=admin

cat >"${tmpdir}/zzzzz.yml" <<EOF
Clusters:
  zzzzz:
    PostgreSQL:
      Connection:
        client_encoding: utf8
        host: ${hostname}
        port: "${pgport}"
        dbname: arvados_test
        user: arvados
        password: insecure_arvados_test
    ManagementToken: e687950a23c3a9bceec28c6223a06c79
    SystemRootToken: systemusertesttoken1234567890aoeuidhtnsqjkxbmwvzpy
    API:
      RequestTimeout: 30s
    TLS:
      Insecure: true
    Collections:
      BlobSigningKey: zfhgfenhffzltr9dixws36j1yhksjoll2grmku38mi7yxd66h5j4q9w4jzanezacp8s6q0ro3hxakfye02152hncy6zml2ed0uc
      TrustAllContent: true
      ForwardSlashNameSubstitution: /
    Services:
      RailsAPI:
        InternalURLs:
          "https://${hostname}:${ARVADOS_TEST_API_HOST##*:}/": {}
      Controller:
        ExternalURL: http://0.0.0.0:9999/
        InternalURLs:
          "http://0.0.0.0:9999/": {}
    SystemLogs:
      LogLevel: debug
EOF
case "${config_method}" in
    pam)
        setup_pam_ldap="apt update && DEBIAN_FRONTEND=noninteractive apt install -y ldap-utils libpam-ldap && pam-auth-update --package /usr/share/pam-configs/ldap"
        cat >>"${tmpdir}/zzzzz.yml" <<EOF
    Login:
      PAM:
        Enable: true
        # Without this specific DefaultEmailDomain, inserted users
        # would prevent subsequent database/reset from working (see
        # database_controller.rb).
        DefaultEmailDomain: example.com
EOF
        ;;
    ldap)
        setup_pam_ldap=""
        cat >>"${tmpdir}/zzzzz.yml" <<EOF
    Login:
      LDAP:
        Enable: true
        URL: ${ldapurl}
        StartTLS: false
        SearchBase: dc=example,dc=org
        SearchBindUser: cn=admin,dc=example,dc=org
        SearchBindPassword: admin
EOF
            ;;
esac

cat >&2 "${tmpdir}/zzzzz.yml"

cat >"${tmpdir}/pam_ldap.conf" <<EOF
base dc=example,dc=org
ldap_version 3
uri ${ldapurl}
pam_password crypt
binddn cn=${adminuser},dc=example,dc=org
bindpw ${adminpassword}
EOF

cat >"${tmpdir}/add_example_user.ldif" <<EOF
dn: cn=bar,dc=example,dc=org
objectClass: posixGroup
objectClass: top
cn: bar
gidNumber: 11111
description: "Example group 'bar'"

dn: uid=foo-bar,dc=example,dc=org
uid: foo-bar
cn: "Foo Bar"
givenName: Foo
sn: Bar
mail: foo-bar-baz@example.com
objectClass: inetOrgPerson
objectClass: posixAccount
objectClass: top
objectClass: shadowAccount
shadowMax: 180
shadowMin: 1
shadowWarning: 7
shadowLastChange: 10701
loginShell: /bin/bash
uidNumber: 11111
gidNumber: 11111
homeDirectory: /home/foo-bar
userPassword: ${passwordhash}
EOF

echo >&2 "Adding example user entry user=foo-bar pass=secret (retrying until server comes up)"
docker run --rm --entrypoint= \
       -v "${tmpdir}/add_example_user.ldif":/add_example_user.ldif:ro \
       osixia/openldap:1.3.0 \
       bash -c "for f in \$(seq 1 5); do if ldapadd -H '${ldapurl}' -D 'cn=${adminuser},dc=example,dc=org' -w '${adminpassword}' -f /add_example_user.ldif; then exit 0; else sleep 2; fi; done; echo 'failed to add user entry'; exit 1"

echo >&2 "Building arvados controller binary to run in container"
go build -o "${tmpdir}" ../../../cmd/arvados-server

ctrlctr=ctrl-${RANDOM}
echo >&2 "Starting arvados controller in docker container ${ctrlctr}"
docker run --detach --rm --name=${ctrlctr} \
       -p 9999 \
       -v "${tmpdir}/pam_ldap.conf":/etc/pam_ldap.conf:ro \
       -v "${tmpdir}/arvados-server":/bin/arvados-server:ro \
       -v "${tmpdir}/zzzzz.yml":/etc/arvados/config.yml:ro \
       -v $(realpath "${PWD}/../../.."):/arvados:ro \
       debian:10 \
       bash -c "${setup_pam_ldap:-true} && arvados-server controller"
docker logs --follow ${ctrlctr} 2>$debug >$debug &
ctrlhostports=$(docker port ${ctrlctr} 9999/tcp)
ctrlport=${ctrlhostports##*:}

echo >&2 "Waiting for arvados controller to come up..."
for f in $(seq 1 20); do
    if curl -s "http://0.0.0.0:${ctrlport}/arvados/v1/config" >/dev/null; then
        break
    else
        sleep 1
    fi
    echo -n >&2 .
done
echo >&2
echo >&2 "Arvados controller is up at http://0.0.0.0:${ctrlport}"

check_contains() {
    resp="${1}"
    str="${2}"
    if ! echo "${resp}" | fgrep -q "${str}"; then
        echo >&2 "${resp}"
        echo >&2 "FAIL: expected in response, but not found: ${str@Q}"
        return 1
    fi
}

set +x

echo >&2 "Testing authentication failure"
resp="$(set -x; curl -s --include -d username=foo-bar -d password=nosecret "http://0.0.0.0:${ctrlport}/arvados/v1/users/authenticate" | tee $debug)"
check_contains "${resp}" "HTTP/1.1 401"
if [[ "${config_method}" = ldap ]]; then
    check_contains "${resp}" '{"errors":["LDAP: Authentication failure (with username \"foo-bar\" and password)"]}'
else
    check_contains "${resp}" '{"errors":["PAM: Authentication failure (with username \"foo-bar\" and password)"]}'
fi

echo >&2 "Testing authentication success"
resp="$(set -x; curl -s --include -d username=foo-bar -d password=secret "http://0.0.0.0:${ctrlport}/arvados/v1/users/authenticate" | tee $debug)"
check_contains "${resp}" "HTTP/1.1 200"
check_contains "${resp}" '"api_token":"'
check_contains "${resp}" '"scopes":["all"]'
check_contains "${resp}" '"uuid":"zzzzz-gj3su-'

secret="${resp##*api_token\":\"}"
secret="${secret%%\"*}"
uuid="${resp##*uuid\":\"}"
uuid="${uuid%%\"*}"
token="v2/$uuid/$secret"
echo >&2 "New token is ${token}"

resp="$(set -x; curl -s --include -H "Authorization: Bearer ${token}" "http://0.0.0.0:${ctrlport}/arvados/v1/users/current" | tee $debug)"
check_contains "${resp}" "HTTP/1.1 200"
if [[ "${config_method}" = ldap ]]; then
    # user fields come from LDAP attributes
    check_contains "${resp}" '"first_name":"Foo"'
    check_contains "${resp}" '"last_name":"Bar"'
    check_contains "${resp}" '"username":"foobar"' # "-" removed by rails api
    check_contains "${resp}" '"email":"foo-bar-baz@example.com"'
else
    # PAMDefaultEmailDomain
    check_contains "${resp}" '"email":"foo-bar@example.com"'
fi

cleanup
