#!/bin/bash

set -e

EXITCODE=0
DEBUG=${ARVADOS_DEBUG:-0}

STDOUT_IF_DEBUG=/dev/null
STDERR_IF_DEBUG=/dev/null
DASHQ_UNLESS_DEBUG=-q
if [[ "$DEBUG" != 0 ]]; then
    STDOUT_IF_DEBUG=/dev/stdout
    STDERR_IF_DEBUG=/dev/stderr
    DASHQ_UNLESS_DEBUG=
fi

case "$TARGET" in
    debian*|ubuntu*)
        FORMAT=deb
        ;;
    centos*)
        FORMAT=rpm
        ;;
    *)
        echo -e "$0: Unknown target '$TARGET'.\n" >&2
        exit 1
        ;;
esac

if ! [[ -n "$WORKSPACE" ]]; then
  echo >&2 "$helpmessage"
  echo >&2
  echo >&2 "Error: WORKSPACE environment variable not set"
  echo >&2
  exit 1
fi

if ! [[ -d "$WORKSPACE" ]]; then
  echo >&2 "$helpmessage"
  echo >&2
  echo >&2 "Error: $WORKSPACE is not a directory"
  echo >&2
  exit 1
fi

title () {
    txt="********** $1 **********"
    printf "\n%*s%s\n\n" $((($COLUMNS-${#txt})/2)) "" "$txt"
}

checkexit() {
    if [[ "$1" != "0" ]]; then
        title "!!!!!! $2 FAILED !!!!!!"
    fi
}


# Find the SSO server package

cd "$WORKSPACE"

if [[ ! -d "/var/www/arvados-sso" ]]; then
  echo "/var/www/arvados-sso should exist"
  exit 1
fi

if [[ ! -e "/etc/arvados/sso/application.yml" ]]; then
    mkdir -p /etc/arvados/sso/
    RANDOM_PASSWORD=`date | md5sum |cut -f1 -d' '`
    cp config/application.yml.example /etc/arvados/sso/application.yml
    sed -i -e 's/uuid_prefix: ~/uuid_prefix: zzzzz/' /etc/arvados/sso/application.yml
    sed -i -e "s/secret_token: ~/secret_token: $RANDOM_PASSWORD/" /etc/arvados/sso/application.yml
fi

if [[ ! -e "/etc/arvados/sso/database.yml" ]]; then
  # We need to set up our database configuration now.
  if [[ "$FORMAT" == "rpm" ]]; then
    service postgresql initdb
    sed -i -e "s/127.0.0.1\/32          ident/127.0.0.1\/32          md5/" /var/lib/pgsql/data/pg_hba.conf
    sed -i -e "s/::1\/128               ident/::1\/128               md5/" /var/lib/pgsql/data/pg_hba.conf
  fi
  service postgresql start

  RANDOM_PASSWORD=`date | md5sum |cut -f1 -d' '`
  cat >/etc/arvados/sso/database.yml <<EOF
production:
  adapter: postgresql
  encoding: utf8
  database: sso_provider_production
  username: sso_provider_user
  password: $RANDOM_PASSWORD
  host: localhost
EOF

  su postgres -c "psql -c \"CREATE USER sso_provider_user WITH PASSWORD '$RANDOM_PASSWORD'\""
  su postgres -c "createdb sso_provider_production -O sso_provider_user"
fi

if [[ "$FORMAT" == "deb" ]]; then
  # Test 2: the package should reconfigure cleanly
  dpkg-reconfigure arvados-sso-server || EXITCODE=3

  cd /var/www/arvados-sso/current/
  /usr/local/rvm/bin/rvm-exec default bundle list >"$ARV_PACKAGES_DIR/arvados-sso-server.gems"

  # Test 3: the package should remove cleanly
  apt-get remove arvados-sso-server --yes || EXITCODE=3

  checkexit $EXITCODE "apt-get remove arvados-sso-server --yes"

  # Test 4: the package configuration should remove cleanly
  dpkg --purge arvados-sso-server || EXITCODE=4

  checkexit $EXITCODE "dpkg --purge arvados-sso-server"

  if [[ -e "/var/www/arvados-sso" ]]; then
    EXITCODE=4
  fi

  checkexit $EXITCODE "leftover items under /var/www/arvados-sso"

  # Test 5: the package should remove cleanly with --purge

  apt-get remove arvados-sso-server --purge --yes || EXITCODE=5

  checkexit $EXITCODE "apt-get remove arvados-sso-server --purge --yes"

  if [[ -e "/var/www/arvados-sso" ]]; then
    EXITCODE=5
  fi

  checkexit $EXITCODE "leftover items under /var/www/arvados-sso"

elif [[ "$FORMAT" == "rpm" ]]; then

  # Set up Nginx first
  # (courtesy of https://www.phusionpassenger.com/library/walkthroughs/deploy/ruby/ownserver/nginx/oss/el6/install_passenger.html)
  yum install -q -y epel-release pygpgme curl
  curl --fail -sSLo /etc/yum.repos.d/passenger.repo https://oss-binaries.phusionpassenger.com/yum/definitions/el-passenger.repo
  yum install -q -y nginx passenger
  sed -i -e 's/^# passenger/passenger/' /etc/nginx/conf.d/passenger.conf
  # Done setting up Nginx

  # Test 2: the package should reinstall cleanly
  yum --assumeyes reinstall arvados-sso-server || EXITCODE=3

  cd /var/www/arvados-sso/current/
  /usr/local/rvm/bin/rvm-exec default bundle list >$ARV_PACKAGES_DIR/arvados-sso-server.gems

  # Test 3: the package should remove cleanly
  yum -q -y remove arvados-sso-server || EXITCODE=3

  checkexit $EXITCODE "yum -q -y remove arvados-sso-server"

  if [[ -e "/var/www/arvados-sso" ]]; then
    EXITCODE=3
  fi

  checkexit $EXITCODE "leftover items under /var/www/arvados-sso"

fi

if [[ "$EXITCODE" == "0" ]]; then
  echo "Testing complete, no errors!"
else
  echo "Errors while testing!"
fi

exit $EXITCODE
