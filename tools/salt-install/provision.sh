#!/bin/bash -x

# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: CC-BY-SA-3.0

# If you want to test arvados in a single host, you can run this script, which
# will install it using salt masterless
# This script is run by the Vagrant file when you run it with
#
# vagrant up

##########################################################
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

set -o pipefail

usage() {
  echo >&2
  echo >&2 "Usage: $0 [-h] [-h]"
  echo >&2
  echo >&2 "$0 options:"
  echo >&2 "  -v, --vagrant           Run in vagrant and use the /vagrant shared dir"
  echo >&2 "  -p <N>, --ssl-port <N>  SSL port to use for the web applications"
  echo >&2 "  -h, --help              Display this help and exit"
  echo >&2
}

arguments() {
  # NOTE: This requires GNU getopt (part of the util-linux package on Debian-based distros).
  TEMP=`getopt -o hvp: \
    --long help,vagrant,ssl-port: \
    -n "$0" -- "$@"`

  if [ $? != 0 ] ; then echo "GNU getopt missing? Use -h for help"; exit 1 ; fi
  # Note the quotes around `$TEMP': they are essential!
  eval set -- "$TEMP"

  while [ $# -ge 1 ]; do
    case $1 in
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

HOST_SSL_PORT=443

arguments $@

# Salt's dir
## states
S_DIR="/srv/salt"
## formulas
F_DIR="/srv/formulas"
##pillars
P_DIR="/srv/pillars"

apt-get update
apt-get install -y curl git

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
    - ${F_DIR}/*/test/salt/states

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
    - example_add_snakeoil_certs
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
for f in postgres arvados nginx docker locale; do
  # git clone https://github.com/saltstack-formulas/${f}-formula.git
  git clone https://github.com/netmanagers/${f}-formula.git
done

if [ "x${BRANCH}" != "x" ]; then
  cd ${F_DIR}/arvados-formula
  git checkout -t origin/${BRANCH}
  cd -
fi

# sed "s/__DOMAIN__/${DOMAIN}/g; s/__CLUSTER__/${CLUSTER}/g; s/__RELEASE__/${RELEASE}/g; s/__VERSION__/${VERSION}/g" \
#   ${CONFIG_DIR}/arvados_dev.sls > ${P_DIR}/arvados.sls

if [ "x${VAGRANT}" = "xyes" ]; then
  SOURCE_PILLARS_DIR="/vagrant/${CONFIG_DIR}"
else
  SOURCE_PILLARS_DIR="./${CONFIG_DIR}"
fi

# Replace cluster and domain name in the example pillars
for f in ${SOURCE_PILLARS_DIR}/*; do
  # sed "s/example.net/${DOMAIN}/g; s/fixme/${CLUSTER}/g" \
  sed "s/__DOMAIN__/${DOMAIN}/g;
       s/__CLUSTER__/${CLUSTER}/g;
       s/__RELEASE__/${RELEASE}/g;
       s/__HOST_SSL_PORT__/${HOST_SSL_PORT}/g;
       s/__GUEST_SSL_PORT__/${GUEST_SSL_PORT}/g;
       s/__INITIAL_USER__/${INITIAL_USER}/g;
       s/__INITIAL_USER_EMAIL__/${INITIAL_USER_EMAIL}/g;
       s/__INITIAL_USER_PASSWORD__/${INITIAL_USER_PASSWORD}/g;
       s/__VERSION__/${VERSION}/g" \
  ${f} > ${P_DIR}/$(basename ${f})
done

# Let's write an /etc/hosts file that points all the hosts to localhost

echo "127.0.0.2 api keep keep0 collections download ws workbench workbench2 ${CLUSTER}.${DOMAIN} api.${CLUSTER}.${DOMAIN} keep.${CLUSTER}.${DOMAIN} keep0.${CLUSTER}.${DOMAIN} collections.${CLUSTER}.${DOMAIN} download.${CLUSTER}.${DOMAIN} ws.${CLUSTER}.${DOMAIN} workbench.${CLUSTER}.${DOMAIN} workbench2.${CLUSTER}.${DOMAIN}" >> /etc/hosts

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
salt-call --local state.apply -l debug

# FIXME! #16992 Temporary fix for psql call in arvados-api-server
if [ "x${DELETE_PSQL}" = "xyes" ]; then
  echo "Removing .psql file"
  rm /root/.psqlrc
fi

if [ "x${RESTORE_PSQL}" = "xyes" ]; then
  echo "Restroting .psql file"
  mv -v /root/.psqlrc.provision.backup /root/.psqlrc
fi
# END FIXME! #16992 Temporary fix for psql call in arvados-api-server
