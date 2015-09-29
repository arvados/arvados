#!/bin/bash

. `dirname "$(readlink -f "$0")"`/run-library.sh

read -rd "\000" helpmessage <<EOF
$(basename $0): Test Arvados SSO package

Syntax:
        WORKSPACE=/path/to/arvados-sso $(basename $0) [options]

Options:

--debug
    Output debug information (default: false)
--target
    Distribution to build packages for (default: debian7)

WORKSPACE=path         Path to the Arvados SSO source tree

EOF

EXITCODE=0
DEBUG=${ARVADOS_DEBUG:-0}
BUILD_BUNDLE_PACKAGES=0
TARGET=debian7
SUDO=/usr/bin/sudo

if [[ ! -e "$SUDO" ]]; then
  SUDO=
fi

PARSEDOPTS=$(getopt --name "$0" --longoptions \
    help,build-bundle-packages,debug,target: \
    -- "" "$@")
if [ $? -ne 0 ]; then
    exit 1
fi

eval set -- "$PARSEDOPTS"
while [ $# -gt 0 ]; do
    case "$1" in
        --help)
            echo >&2 "$helpmessage"
            echo >&2
            exit 1
            ;;
        --target)
            TARGET="$2"; shift
            ;;
        --debug)
            DEBUG=1
            ;;
        --build-bundle-packages)
            BUILD_BUNDLE_PACKAGES=1
            ;;
        --)
            if [ $# -gt 1 ]; then
                echo >&2 "$0: unrecognized argument '$2'. Try: $0 --help"
                exit 1
            fi
            ;;
    esac
    shift
done

STDOUT_IF_DEBUG=/dev/null
STDERR_IF_DEBUG=/dev/null
DASHQ_UNLESS_DEBUG=-q
if [[ "$DEBUG" != 0 ]]; then
    STDOUT_IF_DEBUG=/dev/stdout
    STDERR_IF_DEBUG=/dev/stderr
    DASHQ_UNLESS_DEBUG=
fi

case "$TARGET" in
    debian7)
        FORMAT=deb
        ;;
    debian8)
        FORMAT=deb
        ;;
    ubuntu1204)
        FORMAT=deb
        ;;
    ubuntu1404)
        FORMAT=deb
        ;;
    centos6)
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

SSO_VERSION=$(version_from_git)
PACKAGE_NAME=arvados-sso-server

if [[ "$FORMAT" == "deb" ]]; then
  PACKAGE_PATH=$WORKSPACE/packages/$TARGET/${PACKAGE_NAME}_${SSO_VERSION}_amd64.deb
elif [[ "$FORMAT" == "rpm" ]]; then
  PACKAGE_PATH=$WORKSPACE/packages/$TARGET/${PACKAGE_NAME}-${SSO_VERSION}-1.x86_64.rpm
fi

# Test 1a: the package to test must exist
if [[ ! -f $PACKAGE_PATH ]]; then
  echo "Latest package not found at $PACKAGE_PATH. Please build the package first."
  exit 1
fi

if [[ "$FORMAT" == "deb" ]]; then
  # Test 1b: the system/container where we're running the tests must be clean
  set +e
  dpkg -l |grep $PACKAGE_NAME -q
  if [[ "$?" != "1" ]]; then
    echo "Please make sure the $PACKAGE_NAME package is not installed before running this script"
    exit 1
  fi
  set -e
fi

if [[ -e "/var/www/arvados-sso" ]]; then
  echo "Please make sure /var/www/arvados-sso does not exist before running this script"
  exit 1
fi

# Prepare the machine
if [[ "$FORMAT" == "deb" ]]; then
  $SUDO apt-get update
elif [[ "$FORMAT" == "rpm" ]]; then
  $SUDO yum check-update
fi
$SUDO mkdir -p /etc/arvados/sso

if [[ ! -e "/etc/arvados/sso/application.yml" ]]; then
  RANDOM_PASSWORD=`date | md5sum |cut -f1 -d' '`
  cp config/application.yml.example /etc/arvados/sso/application.yml
  sed -i -e 's/uuid_prefix: ~/uuid_prefix: zzzzz/' /etc/arvados/sso/application.yml
  sed -i -e "s/secret_token: ~/secret_token: $RANDOM_PASSWORD/" /etc/arvados/sso/application.yml
fi

if [[ ! -e "/etc/arvados/sso/database.yml" ]]; then
  # We haven't installed our dependencies yet, but we need to set up our
  # database configuration now. Install postgresql if need be.
  if [[ "$FORMAT" == "deb" ]]; then
    install_package postgresql
  elif [[ "$FORMAT" == "rpm" ]]; then
    install_package postgresql-server
    # postgres packaging on CentOS6 is kind of primitive, needs an initdb
    $SUDO service postgresql initdb
    if [ "$TARGET" = "centos6" ]; then
      sed -i -e "s/127.0.0.1\/32          ident/127.0.0.1\/32          md5/" /var/lib/pgsql/data/pg_hba.conf
      sed -i -e "s/::1\/128               ident/::1\/128               md5/" /var/lib/pgsql/data/pg_hba.conf
    fi
  fi
  $SUDO service postgresql start

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
  if [[ "$SUDO" != '' ]]; then
    $SUDO -u postgres psql -c "CREATE USER sso_provider_user WITH PASSWORD '$RANDOM_PASSWORD'"
    $SUDO -u postgres createdb sso_provider_production -O sso_provider_user
  else
    install_package sudo
    if [ "$TARGET" = "centos6" ]; then
      # Work around silly CentOS6 default, cf. https://bugzilla.redhat.com/show_bug.cgi?id=1020147
      sed -i -e 's/Defaults    requiretty/#Defaults    requiretty/' /etc/sudoers
    fi
    /usr/bin/sudo -u postgres psql -c "CREATE USER sso_provider_user WITH PASSWORD '$RANDOM_PASSWORD'"
    /usr/bin/sudo -u postgres createdb sso_provider_production -O sso_provider_user
  fi
fi

if [[ "$FORMAT" == "deb" ]]; then
  # Test 2: the package should install cleanly
  # In order to get the package dependencies, we need to first do an install
  # with dpkg that will fail, then run apt-get to install the dependencies,
  # and the subsequent dpkg installation should succeed.
  set +e
  $SUDO dpkg -i $PACKAGE_PATH > /dev/null 2>&1
  $SUDO apt-get -f install --yes
  set -e
  $SUDO dpkg -i $PACKAGE_PATH || EXITCODE=2

  checkexit $EXITCODE "dpkg -i $PACKAGE_PATH"

  # Test 3: the package should remove cleanly
  $SUDO apt-get remove $PACKAGE_NAME --yes || EXITCODE=3

  checkexit $EXITCODE "apt-get remove $PACKAGE_PATH --yes"

  # Test 4: the package configuration should remove cleanly
  $SUDO dpkg --purge $PACKAGE_NAME || EXITCODE=4

  checkexit $EXITCODE "dpkg --purge $PACKAGE_PATH"

  if [[ -e "/var/www/arvados-sso" ]]; then
    EXITCODE=4
  fi

  checkexit $EXITCODE "leftover items under /var/www/arvados-sso"

  # Test 5: the package should remove cleanly with --purge
  $SUDO dpkg -i $PACKAGE_PATH || EXITCODE=5

  checkexit $EXITCODE "dpkg -i $PACKAGE_PATH"

  $SUDO apt-get remove $PACKAGE_NAME --purge --yes || EXITCODE=5

  checkexit $EXITCODE "apt-get remove $PACKAGE_PATH --purge --yes"

  if [[ -e "/var/www/arvados-sso" ]]; then
    EXITCODE=5
  fi

  checkexit $EXITCODE "leftover items under /var/www/arvados-sso"

elif [[ "$FORMAT" == "rpm" ]]; then

  # Set up Nginx first
  # (courtesy of https://www.phusionpassenger.com/library/walkthroughs/deploy/ruby/ownserver/nginx/oss/el6/install_passenger.html)
  $SUDO yum install -q -y epel-release pygpgme curl
  $SUDO curl --fail -sSLo /etc/yum.repos.d/passenger.repo https://oss-binaries.phusionpassenger.com/yum/definitions/el-passenger.repo
  $SUDO yum install -q -y nginx passenger
  $SUDO sed -i -e 's/^# passenger/passenger/' /etc/nginx/conf.d/passenger.conf
  # Done setting up Nginx

  # Test 2: the package should install cleanly
  $SUDO yum -q -y --nogpgcheck localinstall $PACKAGE_PATH || EXITCODE=3

  checkexit $EXITCODE "yum -q -y --nogpgcheck localinstall $PACKAGE_PATH"

  # Test 3: the package should remove cleanly
  $SUDO yum -q -y remove $PACKAGE_NAME || EXITCODE=3

  checkexit $EXITCODE "yum -q -y remove $PACKAGE_PATH"

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
