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

install_package() {
  PACKAGES=$@
  if [[ "$FORMAT" == "deb" ]]; then
    $SUDO apt-get install $PACKAGES --yes
  elif [[ "$FORMAT" == "rpm" ]]; then
    $SUDO yum -q -y install $PACKAGES
  fi
}

# Find the SSO server package

cd "$WORKSPACE"

SSO_VERSION=$(version_from_git)
PACKAGE_NAME=arvados-sso

if [[ "$FORMAT" == "deb" ]]; then
  PACKAGE_PATH=$WORKSPACE/packages/$TARGET/arvados-sso_${SSO_VERSION}_amd64.deb
elif [[ "$FORMAT" == "rpm" ]]; then
  PACKAGE_PATH=$WORKSPACE/packages/$TARGET/arvados-sso-${SSO_VERSION}.x86_64.rpm
fi

# Test 1a: the package to test must exist
if [[ ! -f $PACKAGE_PATH ]]; then
  echo "Latest package not found at $PACKAGE_PATH. Please build the package first."
  exit 1
fi

if [[ "$FORMAT" == "deb" ]]; then
  # Test 1b: the system/container where we're running the tests must be clean
  set +e
  dpkg -l |grep arvados-sso -q
  if [[ "$?" != "1" ]]; then
    echo "Please make sure the arvados-sso package is not installed before running this script"
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
  install_package postgresql
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
  $SUDO dpkg -i $PACKAGE_PATH

  # Test 3: the package should remove cleanly
  $SUDO apt-get remove arvados-sso --yes

  # Test 4: the package configuration should remove cleanly
  $SUDO dpkg --purge arvados-sso

  if [[ -e "/var/www/arvados-sso" ]]; then
    echo "Error: leftover items under /var/www/arvados-sso."
    exit 4
  fi

  # Test 5: the package should remove cleanly with --purge
  $SUDO dpkg -i $PACKAGE_PATH
  $SUDO apt-get remove arvados-sso --purge --yes

  if [[ -e "/var/www/arvados-sso" ]]; then
    echo "Error: leftover items under /var/www/arvados-sso."
    exit 5
  fi

elif [[ "$FORMAT" == "rpm" ]]; then
  # Test 2: the package should install cleanly
  $SUDO yum -q -y --nogpgcheck localinstall $PACKAGE_PATH

  # Test 3: the package should remove cleanly
  $SUDO yum -q -y remove arvados-sso

  if [[ -e "/var/www/arvados-sso" ]]; then
    echo "Error: leftover items under /var/www/arvados-sso."
    exit 3
  fi

fi

exit $EXITCODE
