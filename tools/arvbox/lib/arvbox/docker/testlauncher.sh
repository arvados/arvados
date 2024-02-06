#!/bin/sh
# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

set -xeu

PGVERSION=$(psql --version | grep -E -o '[0-9]+' | head -n1)

make-ssl-cert generate-default-snakeoil --force-overwrite

chown -R postgres:postgres /etc/ssl/private
chmod 0600 /etc/ssl/private/ssl-cert-snakeoil.key

su postgres -c "/usr/lib/postgresql/$PGVERSION/bin/postgres -D /var/lib/postgresql/$PGVERSION/main -c config_file=/etc/postgresql/$PGVERSION/main/postgresql.conf" &

database_pw=insecure_test_password

sleep 1

su postgres -c "psql postgres -c \"create user arvados with password '$database_pw'\""
su postgres -c "psql postgres -c \"ALTER USER arvados WITH SUPERUSER;\""

ARVADOS_CONTAINER_PATH=/var/lib/arvados-arvbox

mkdir -p $ARVADOS_CONTAINER_PATH/run_tests
cat >$ARVADOS_CONTAINER_PATH/run_tests/config.yml <<EOF
Clusters:
  zzzzz:
    PostgreSQL:
      Connection:
        host: localhost
        user: arvados
        password: ${database_pw}
        dbname: arvados_test
        client_encoding: utf8
EOF

useradd --home-dir /home/arvbox --create-home arvbox

su arvbox -c env
su arvbox -c "ls -l /home"

export CONFIGSRC=$ARVADOS_CONTAINER_PATH/run_tests

while ! su arvbox -c "psql postgres -c\\\du >/dev/null 2>/dev/null" ; do
    sleep 1
done

export WORKSPACE=/usr/src/arvados

exec su arvbox -c "/usr/src/arvados/build/run-tests.sh --temp /home/arvbox $@"
