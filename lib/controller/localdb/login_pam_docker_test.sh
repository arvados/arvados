#!/bin/bash

# This script demonstrates using LDAP for Arvados user authentication.
#
# It configures pam_ldap(5) and arvados controller in a docker
# container, with pam_ldap configured to authenticate against an
# OpenLDAP server in a second docker container.
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

ldapctr=ldap-${RANDOM}
echo >&2 "Starting ldap server in docker container ${ldapctr}"
docker run --rm --detach \
       -p 389 -p 636 \
       --name=${ldapctr} \
       osixia/openldap:1.3.0
docker logs --follow ${ldapctr} 2>$debug >$debug &
ldaphostport=$(docker port ${ldapctr} 389/tcp)
ldapport=${ldaphostport##*:}
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
    Login:
      PAM: true
    SystemLogs:
      LogLevel: debug
EOF

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

dn: uid=foo,dc=example,dc=org
uid: foo
cn: foo
givenName: Foo
sn: Bar
mail: foobar@example.org
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
homeDirectory: /home/foo
userPassword: ${passwordhash}
EOF

echo >&2 "Adding example user entry user=foo pass=secret (retrying until server comes up)"
docker run --rm --entrypoint= \
       -v "${tmpdir}/add_example_user.ldif":/add_example_user.ldif:ro \
       osixia/openldap:1.3.0 \
       bash -c "for f in \$(seq 1 5); do if ldapadd -H '${ldapurl}' -D 'cn=${adminuser},dc=example,dc=org' -w '${adminpassword}' -f /add_example_user.ldif; then exit 0; else sleep 2; fi; done; echo 'failed to add user entry'; exit 1"

ctrlctr=ctrl-${RANDOM}
echo >&2 "Starting arvados controller in docker container ${ctrlctr}"
docker run --detach --rm --name=${ctrlctr} \
       -p 9999 \
       -v "${tmpdir}/pam_ldap.conf":/etc/pam_ldap.conf:ro \
       -v "${GOPATH:-${HOME}/go}/bin/arvados-server":/bin/arvados-server:ro \
       -v "${tmpdir}/zzzzz.yml":/etc/arvados/config.yml:ro \
       -v $(realpath "${PWD}/../../.."):/arvados:ro \
       debian:10 \
       bash -c "apt update && DEBIAN_FRONTEND=noninteractive apt install -y ldap-utils libpam-ldap && pam-auth-update --package /usr/share/pam-configs/ldap && arvados-server controller"
docker logs --follow ${ctrlctr} 2>$debug >$debug &
ctrlhostport=$(docker port ${ctrlctr} 9999/tcp)

echo >&2 "Waiting for arvados controller to come up..."
for f in $(seq 1 20); do
    if curl -s "http://${ctrlhostport}/arvados/v1/config" >/dev/null; then
        break
    else
        sleep 1
    fi
    echo -n >&2 .
done
echo >&2
echo >&2 "Arvados controller is up at http://${ctrlhostport}"

echo >&2 "Testing authentication failure"
curl -s -H "X-Http-Method-Override: GET" -d username=foo -d password=nosecret "http://${ctrlhostport}/login" | tee $debug | grep "Authentication failure"
echo >&2 "Testing authentication success"
curl -s -H "X-Http-Method-Override: GET" -d username=foo -d password=secret "http://${ctrlhostport}/login" | tee $debug | fgrep '{"token":"v2/zzzzz-gj3su-'

cleanup
