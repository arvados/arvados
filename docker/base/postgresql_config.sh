#! /bin/sh

# Configure postgresql in a docker instance.

# Make sure our environment is set appropriately.
[ "$POSTGRES_ROOT_PW" ]   || (echo "POSTGRES_ROOT_PW not set, failing"; exit 1)
[ "$ARVADOS_DEV_USER" ]   || (echo "ARVADOS_DEV_USER not set, failing"; exit 1)
[ "$ARVADOS_DEV_PW" ]     || (echo "ARVADOS_DEV_PW not set, failing"; exit 1)
[ "$ARVADOS_DEV_DB" ]     || (echo "ARVADOS_DEV_DB not set, failing"; exit 1)
[ "$ARVADOS_TEST_USER" ]  || (echo "ARVADOS_TEST_USER not set, failing"; exit 1)
[ "$ARVADOS_TEST_PW" ]    || (echo "ARVADOS_TEST_PW not set, failing"; exit 1)
[ "$ARVADOS_TEST_DB" ]    || (echo "ARVADOS_TEST_DB not set, failing"; exit 1)
[ "$ARVADOS_PROD_USER" ]  || (echo "ARVADOS_PROD_USER not set, failing"; exit 1)
[ "$ARVADOS_PROD_PW" ]    || (echo "ARVADOS_PROD_PW not set, failing"; exit 1)
[ "$ARVADOS_PROD_DB" ]    || (echo "ARVADOS_PROD_DB not set, failing"; exit 1)

/bin/su postgres -c '/usr/lib/postgresql/9.1/bin/postgres --single -D /var/lib/postgresql/9.1/main -c config_file=/etc/postgresql/9.1/main/postgresql.conf' <<EOF
alter role postgres with encrypted password '${POSTGRES_ROOT_PW}';

create user ${ARVADOS_DEV_USER} with encrypted password '${ARVADOS_DEV_PW}';
create database ${ARVADOS_DEV_DB} with owner ${ARVADOS_DEV_USER};

create user ${ARVADOS_TEST_USER} with encrypted password '${ARVADOS_TEST_USER}';
create database ${ARVADOS_TEST_DB} with owner ${ARVADOS_TEST_USER};

create user ${ARVADOS_PROD_USER} with encrypted password '${ARVADOS_PROD_USER}';
create database ${ARVADOS_PROD_DB} with owner ${ARVADOS_PROD_USER};
EOF
