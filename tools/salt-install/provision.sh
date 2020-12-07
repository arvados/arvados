#!/bin/bash

# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: CC-BY-SA-3.0

# If you want to test arvados in a single host, you can run this script, which
# will install it using salt masterless
# This script is run by the Vagrant file when you run it with
#
# vagrant up

##########################################################
# This section are the basic parameters to configure the installation

# The 5 letters name you want to give your cluster
CLUSTER="arva2"
DOMAIN="arv.local"

INITIAL_USER="admin"

# If not specified, the initial user email will be composed as
# INITIAL_USER@CLUSTER.DOMAIN
INITIAL_USER_EMAIL="${INITIAL_USER}@${CLUSTER}.${DOMAIN}"
INITIAL_USER_PASSWORD="password"

# The example config you want to use. Currently, only "single_host" is
# available
CONFIG_DIR="single_host"

# Which release of Arvados repo you want to use
RELEASE="production"
# Which version of Arvados you want to install. Defaults to 'latest'
# in the desired repo
VERSION="latest"

# Host SSL port where you want to point your browser to access Arvados
# Defaults to 443 for regular runs, and to 8443 when called in Vagrant.
# You can point it to another port if desired
# In Vagrant, make sure it matches what you set in the Vagrantfile
# HOST_SSL_PORT=443

# This is a arvados-formula setting.
# If branch is set, the script will switch to it before running salt
# Usually not needed, only used for testing
# BRANCH="master"

##########################################################
# Usually there's no need to modify things below this line

# Formulas versions
POSTGRES_TAG="v0.41.3"
NGINX_TAG="v2.4.0"
DOCKER_TAG="v1.0.0"
LOCALE_TAG="v0.3.4"

set -o pipefail

# capture the directory that the script is running from
SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" >/dev/null 2>&1 && pwd )"

usage() {
  echo >&2
  echo >&2 "Usage: ${0} [-h] [-h]"
  echo >&2
  echo >&2 "${0} options:"
  echo >&2 "  -d, --debug             Run salt installation in debug mode"
  echo >&2 "  -p <N>, --ssl-port <N>  SSL port to use for the web applications"
  echo >&2 "  -t, --test              Test installation running a CWL workflow"
  echo >&2 "  -h, --help              Display this help and exit"
  echo >&2 "  -v, --vagrant           Run in vagrant and use the /vagrant shared dir"
  echo >&2
}

arguments() {
  # NOTE: This requires GNU getopt (part of the util-linux package on Debian-based distros).
  TEMP=$(getopt -o dhp:tv \
    --long debug,help,ssl-port:,test,vagrant \
    -n "${0}" -- "${@}")

  if [ ${?} != 0 ] ; then echo "GNU getopt missing? Use -h for help"; exit 1 ; fi
  # Note the quotes around `$TEMP': they are essential!
  eval set -- "$TEMP"

  while [ ${#} -ge 1 ]; do
    case ${1} in
      -d | --debug)
        LOG_LEVEL="debug"
        shift
        ;;
      -t | --test)
        TEST="yes"
        shift
        ;;
      -v | --vagrant)
        VAGRANT="yes"
        shift
        ;;
      -p | --ssl-port)
        HOST_SSL_PORT=${2}
        shift 2
        ;;
      --)
        shift
        break
        ;;
      *)
        usage
        exit 1
        ;;
    esac
  done
}

LOG_LEVEL="info"
HOST_SSL_PORT=443
TESTS_DIR="tests"

arguments ${@}

# Salt's dir
## states
S_DIR="/srv/salt"
## formulas
F_DIR="/srv/formulas"
##pillars
P_DIR="/srv/pillars"

apt-get update
apt-get install -y curl git jq

dpkg -l |grep salt-minion
if [ ${?} -eq 0 ]; then
  echo "Salt already installed"
else
  curl -L https://bootstrap.saltstack.com -o /tmp/bootstrap_salt.sh
  sh /tmp/bootstrap_salt.sh -XUdfP -x python3
  /bin/systemctl disable salt-minion.service
fi

# Set salt to masterless mode
cat > /etc/salt/minion << EOFSM
file_client: local
file_roots:
  base:
    - ${S_DIR}
    - ${F_DIR}/*
    - ${F_DIR}/*/test/salt/states/examples

pillar_roots:
  base:
    - ${P_DIR}
EOFSM

mkdir -p ${S_DIR}
mkdir -p ${F_DIR}
mkdir -p ${P_DIR}

# States
cat > ${S_DIR}/top.sls << EOFTSLS
base:
  '*':
    - single_host.host_entries
    - single_host.snakeoil_certs
    - locale
    - nginx.passenger
    - postgres
    - docker
    - arvados
EOFTSLS

# Pillars
cat > ${P_DIR}/top.sls << EOFPSLS
base:
  '*':
    - arvados
    - docker
    - locale
    - nginx_api_configuration
    - nginx_controller_configuration
    - nginx_keepproxy_configuration
    - nginx_keepweb_configuration
    - nginx_passenger
    - nginx_websocket_configuration
    - nginx_webshell_configuration
    - nginx_workbench2_configuration
    - nginx_workbench_configuration
    - postgresql
EOFPSLS

# Get the formula and dependencies
cd ${F_DIR} || exit 1
git clone https://github.com/netmanagers/arvados-formula.git
git clone --branch "${DOCKER_TAG}" https://github.com/saltstack-formulas/docker-formula.git
git clone --branch "${LOCALE_TAG}" https://github.com/saltstack-formulas/locale-formula.git
git clone --branch "${NGINX_TAG}" https://github.com/saltstack-formulas/nginx-formula.git
git clone --branch "${POSTGRES_TAG}" https://github.com/saltstack-formulas/postgres-formula.git

if [ "x${BRANCH}" != "x" ]; then
  cd ${F_DIR}/arvados-formula || exit 1
  git checkout -t origin/"${BRANCH}"
  cd -
fi

if [ "x${VAGRANT}" = "xyes" ]; then
  SOURCE_PILLARS_DIR="/vagrant/${CONFIG_DIR}"
  TESTS_DIR="/vagrant/${TESTS_DIR}"
else
  SOURCE_PILLARS_DIR="${SCRIPT_DIR}/${CONFIG_DIR}"
  TESTS_DIR="${SCRIPT_DIR}/${TESTS_DIR}"
fi

# Replace cluster and domain name in the example pillars and test files
for f in "${SOURCE_PILLARS_DIR}"/*; do
  sed "s/__CLUSTER__/${CLUSTER}/g;
       s/__DOMAIN__/${DOMAIN}/g;
       s/__RELEASE__/${RELEASE}/g;
       s/__HOST_SSL_PORT__/${HOST_SSL_PORT}/g;
       s/__GUEST_SSL_PORT__/${GUEST_SSL_PORT}/g;
       s/__INITIAL_USER__/${INITIAL_USER}/g;
       s/__INITIAL_USER_EMAIL__/${INITIAL_USER_EMAIL}/g;
       s/__INITIAL_USER_PASSWORD__/${INITIAL_USER_PASSWORD}/g;
       s/__VERSION__/${VERSION}/g" \
  "${f}" > "${P_DIR}"/$(basename "${f}")
done

mkdir -p /tmp/cluster_tests
# Replace cluster and domain name in the example pillars and test files
for f in "${TESTS_DIR}"/*; do
  sed "s/__CLUSTER__/${CLUSTER}/g;
       s/__DOMAIN__/${DOMAIN}/g;
       s/__HOST_SSL_PORT__/${HOST_SSL_PORT}/g;
       s/__INITIAL_USER__/${INITIAL_USER}/g;
       s/__INITIAL_USER_EMAIL__/${INITIAL_USER_EMAIL}/g;
       s/__INITIAL_USER_PASSWORD__/${INITIAL_USER_PASSWORD}/g" \
  ${f} > /tmp/cluster_tests/$(basename ${f})
done
chmod 755 /tmp/cluster_tests/run-test.sh

# FIXME! #16992 Temporary fix for psql call in arvados-api-server
if [ -e /root/.psqlrc ]; then
  if ! ( grep 'pset pager off' /root/.psqlrc ); then
    RESTORE_PSQL="yes"
    cp /root/.psqlrc /root/.psqlrc.provision.backup
  fi
else
  DELETE_PSQL="yes"
fi

echo '\pset pager off' >> /root/.psqlrc
# END FIXME! #16992 Temporary fix for psql call in arvados-api-server

# Now run the install
salt-call --local state.apply -l ${LOG_LEVEL}

# FIXME! #16992 Temporary fix for psql call in arvados-api-server
if [ "x${DELETE_PSQL}" = "xyes" ]; then
  echo "Removing .psql file"
  rm /root/.psqlrc
fi

if [ "x${RESTORE_PSQL}" = "xyes" ]; then
  echo "Restoring .psql file"
  mv -v /root/.psqlrc.provision.backup /root/.psqlrc
fi
# END FIXME! #16992 Temporary fix for psql call in arvados-api-server

# Leave a copy of the Arvados CA so the user can copy it where it's required
echo "Copying the Arvados CA certificate to the installer dir, so you can import it"
# If running in a vagrant VM, also add default user to docker group
if [ "x${VAGRANT}" = "xyes" ]; then
  cp /etc/ssl/certs/arvados-snakeoil-ca.pem /vagrant

  echo "Adding the vagrant user to the docker group"
  usermod -a -G docker vagrant
else
  cp /etc/ssl/certs/arvados-snakeoil-ca.pem ${SCRIPT_DIR}
fi

# Test that the installation finished correctly
if [ "x${TEST}" = "xyes" ]; then
  cd /tmp/cluster_tests
  ./run-test.sh
fi
