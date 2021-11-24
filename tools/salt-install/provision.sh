#!/bin/bash

# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: CC-BY-SA-3.0

# If you want to test arvados in a single host, you can run this script, which
# will install it using salt masterless
# This script is run by the Vagrant file when you run it with
#
# vagrant up

set -o pipefail

# capture the directory that the script is running from
SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" >/dev/null 2>&1 && pwd )"

usage() {
  echo >&2
  echo >&2 "Usage: ${0} [-h] [-h]"
  echo >&2
  echo >&2 "${0} options:"
  echo >&2 "  -d, --debug                                 Run salt installation in debug mode"
  echo >&2 "  -c <local.params>, --config <local.params>  Path to the local.params config file"
  echo >&2 "  -t, --test                                  Test installation running a CWL workflow"
  echo >&2 "  -r, --roles                                 List of Arvados roles to apply to the host, comma separated"
  echo >&2 "                                              Possible values are:"
  echo >&2 "                                                api"
  echo >&2 "                                                controller"
  echo >&2 "                                                dispatcher"
  echo >&2 "                                                keepproxy"
  echo >&2 "                                                keepstore"
  echo >&2 "                                                keepweb"
  echo >&2 "                                                shell"
  echo >&2 "                                                webshell"
  echo >&2 "                                                websocket"
  echo >&2 "                                                workbench"
  echo >&2 "                                                workbench2"
  echo >&2 "                                              Defaults to applying them all"
  echo >&2 "  -h, --help                                  Display this help and exit"
  echo >&2 "  --dump-config <dest_dir>                    Dumps the pillars and states to a directory"
  echo >&2 "                                              This parameter does not perform any installation at all. It's"
  echo >&2 "                                              intended to give you a parsed sot of configuration files so"
  echo >&2 "                                              you can inspect them or use them in you Saltstack infrastructure."
  echo >&2 "                                              It"
  echo >&2 "                                                - parses the pillar and states templates,"
  echo >&2 "                                                - downloads the helper formulas with their desired versions,"
  echo >&2 "                                                - prepares the 'top.sls' files both for pillars and states"
  echo >&2 "                                                  for the selected role/s"
  echo >&2 "                                                - writes the resulting files into <dest_dir>"
  echo >&2 "  -v, --vagrant                               Run in vagrant and use the /vagrant shared dir"
  echo >&2 "  --development                               Run in dev mode, using snakeoil certs"
  echo >&2
}

arguments() {
  # NOTE: This requires GNU getopt (part of the util-linux package on Debian-based distros).
  if ! which getopt > /dev/null; then
    echo >&2 "GNU getopt is required to run this script. Please install it and re-reun it"
    exit 1
  fi

  TEMP=$(getopt -o c:dhp:r:tv \
    --long config:,debug,development,dump-config:,help,roles:,test,vagrant \
    -n "${0}" -- "${@}")

  if [ ${?} != 0 ];
    then echo "Please check the parameters you entered and re-run again"
    exit 1
  fi
  # Note the quotes around `$TEMP': they are essential!
  eval set -- "$TEMP"

  while [ ${#} -ge 1 ]; do
    case ${1} in
      -c | --config)
        CONFIG_FILE=${2}
        shift 2
        ;;
      -d | --debug)
        LOG_LEVEL="debug"
        shift
        set -x
        ;;
      --dump-config)
        if [[ ${2} = /* ]]; then
          DUMP_SALT_CONFIG_DIR=${2}
        else
          DUMP_SALT_CONFIG_DIR=${PWD}/${2}
        fi
        ## states
        S_DIR="${DUMP_SALT_CONFIG_DIR}/salt"
        ## formulas
        F_DIR="${DUMP_SALT_CONFIG_DIR}/formulas"
        ## pillars
        P_DIR="${DUMP_SALT_CONFIG_DIR}/pillars"
        ## tests
        T_DIR="${DUMP_SALT_CONFIG_DIR}/tests"
        DUMP_CONFIG="yes"
        shift 2
        ;;
      --development)
        DEV_MODE="yes"
        shift 1
        ;;
      -r | --roles)
        for i in ${2//,/ }
          do
            # Verify the role exists
            if [[ ! "database,api,controller,keepstore,websocket,keepweb,workbench2,webshell,keepproxy,shell,workbench,dispatcher" == *"$i"* ]]; then
              echo "The role '${i}' is not a valid role"
              usage
              exit 1
            fi
            ROLES="${ROLES} ${i}"
          done
          shift 2
        ;;
      -t | --test)
        TEST="yes"
        shift
        ;;
      -v | --vagrant)
        VAGRANT="yes"
        shift
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

DEV_MODE="no"
CONFIG_FILE="${SCRIPT_DIR}/local.params"
CONFIG_DIR="local_config_dir"
DUMP_CONFIG="no"
LOG_LEVEL="info"
CONTROLLER_EXT_SSL_PORT=443
TESTS_DIR="tests"

CLUSTER=""
DOMAIN=""

# Hostnames/IPs used for single-host deploys
HOSTNAME_EXT=""
HOSTNAME_INT="127.0.1.1"

# Initial user setup
INITIAL_USER=""
INITIAL_USER_EMAIL=""
INITIAL_USER_PASSWORD=""

CONTROLLER_EXT_SSL_PORT=8000
KEEP_EXT_SSL_PORT=25101
# Both for collections and downloads
KEEPWEB_EXT_SSL_PORT=9002
WEBSHELL_EXT_SSL_PORT=4202
WEBSOCKET_EXT_SSL_PORT=8002
WORKBENCH1_EXT_SSL_PORT=443
WORKBENCH2_EXT_SSL_PORT=3001

USE_LETSENCRYPT="no"
CUSTOM_CERTS_DIR="./certs"

## These are ARVADOS-related parameters
# For a stable release, change RELEASE "production" and VERSION to the
# package version (including the iteration, e.g. X.Y.Z-1) of the
# release.
# The "local.params.example.*" files already set "RELEASE=production"
# to deploy  production-ready packages
RELEASE="production"
VERSION="2.3.1-1"

# These are arvados-formula-related parameters
# An arvados-formula tag. For a stable release, this should be a
# branch name (e.g. X.Y-dev) or tag for the release.
# ARVADOS_TAG="2.3.0"
BRANCH="2.3-dev"

# Other formula versions we depend on
POSTGRES_TAG="v0.41.6"
NGINX_TAG="temp-fix-missing-statements-in-pillar"
DOCKER_TAG="v2.0.7"
LOCALE_TAG="v0.3.4"
LETSENCRYPT_TAG="v2.1.0"

# Salt's dir
DUMP_SALT_CONFIG_DIR=""
## states
S_DIR="/srv/salt"
## formulas
F_DIR="/srv/formulas"
## pillars
P_DIR="/srv/pillars"
## tests
T_DIR="/tmp/cluster_tests"

arguments ${@}

if [ -s ${CONFIG_FILE} ]; then
  source ${CONFIG_FILE}
else
  echo >&2 "You don't seem to have a config file with initial values."
  echo >&2 "Please create a '${CONFIG_FILE}' file as described in"
  echo >&2 "  * https://doc.arvados.org/install/salt-single-host.html#single_host, or"
  echo >&2 "  * https://doc.arvados.org/install/salt-multi-host.html#multi_host_multi_hostnames"
  exit 1
fi

if [ ! -d ${CONFIG_DIR} ]; then
  echo >&2 "You don't seem to have a config directory with pillars and states."
  echo >&2 "Please create a '${CONFIG_DIR}' directory (as configured in your '${CONFIG_FILE}'). Please see"
  echo >&2 "  * https://doc.arvados.org/install/salt-single-host.html#single_host, or"
  echo >&2 "  * https://doc.arvados.org/install/salt-multi-host.html#multi_host_multi_hostnames"
  exit 1
fi

if grep -q 'fixme_or_this_wont_work' ${CONFIG_FILE} ; then
  echo >&2 "The config file ${CONFIG_FILE} has some parameters that need to be modified."
  echo >&2 "Please, fix them and re-run the provision script."
  exit 1
fi

if ! grep -qE '^[[:alnum:]]{5}$' <<<${CLUSTER} ; then
  echo >&2 "ERROR: <CLUSTER> must be exactly 5 alphanumeric characters long"
  echo >&2 "Fix the cluster name in the 'local.params' file and re-run the provision script"
  exit 1
fi

# Only used in single_host/single_name deploys
if [ "x${HOSTNAME_EXT}" = "x" ] ; then
  HOSTNAME_EXT="${CLUSTER}.${DOMAIN}"
fi

if [ "${DUMP_CONFIG}" = "yes" ]; then
  echo "The provision installer will just dump a config under ${DUMP_SALT_CONFIG_DIR} and exit"
else
  # Install a few dependency packages
  # First, let's figure out the OS we're working on
  OS_ID=$(grep ^ID= /etc/os-release |cut -f 2 -d=  |cut -f 2 -d \")
  echo "Detected distro: ${OS_ID}"

  case ${OS_ID} in
    "centos")
      echo "WARNING! Disabling SELinux, see https://dev.arvados.org/issues/18019"
      sed -i 's/SELINUX=enforcing/SELINUX=permissive' /etc/sysconfig/selinux
      setenforce permissive
      yum install -y  curl git jq
      ;;
    "debian"|"ubuntu")
      DEBIAN_FRONTEND=noninteractive apt update
      DEBIAN_FRONTEND=noninteractive apt install -y curl git jq
      ;;
  esac

  if which salt-call; then
    echo "Salt already installed"
  else
    curl -L https://bootstrap.saltstack.com -o /tmp/bootstrap_salt.sh
    sh /tmp/bootstrap_salt.sh -XdfP -x python3
    /bin/systemctl stop salt-minion.service
    /bin/systemctl disable salt-minion.service
  fi

  # Set salt to masterless mode
  cat > /etc/salt/minion << EOFSM
failhard: "True"

file_client: local
file_roots:
  base:
    - ${S_DIR}
    - ${F_DIR}/*

pillar_roots:
  base:
    - ${P_DIR}
EOFSM
fi

mkdir -p ${S_DIR} ${F_DIR} ${P_DIR} ${T_DIR}

# Get the formula and dependencies
cd ${F_DIR} || exit 1
echo "Cloning formulas"
rm -rf ${F_DIR}/* || exit 1
git clone --quiet https://github.com/saltstack-formulas/docker-formula.git ${F_DIR}/docker
( cd docker && git checkout --quiet tags/"${DOCKER_TAG}" -b "${DOCKER_TAG}" )

git clone --quiet https://github.com/saltstack-formulas/locale-formula.git ${F_DIR}/locale
( cd locale && git checkout --quiet tags/"${LOCALE_TAG}" -b "${LOCALE_TAG}" )

git clone --quiet https://github.com/netmanagers/nginx-formula.git ${F_DIR}/nginx
( cd nginx && git checkout --quiet tags/"${NGINX_TAG}" -b "${NGINX_TAG}" )

git clone --quiet https://github.com/saltstack-formulas/postgres-formula.git ${F_DIR}/postgres
( cd postgres && git checkout --quiet tags/"${POSTGRES_TAG}" -b "${POSTGRES_TAG}" )

git clone --quiet https://github.com/saltstack-formulas/letsencrypt-formula.git ${F_DIR}/letsencrypt
( cd letsencrypt && git checkout --quiet tags/"${LETSENCRYPT_TAG}" -b "${LETSENCRYPT_TAG}" )

git clone --quiet https://git.arvados.org/arvados-formula.git ${F_DIR}/arvados

# If we want to try a specific branch of the formula
if [ "x${BRANCH}" != "x" ]; then
  ( cd ${F_DIR}/arvados && git checkout --quiet -t origin/"${BRANCH}" -b "${BRANCH}" )
elif [ "x${ARVADOS_TAG}" != "x" ]; then
( cd ${F_DIR}/arvados && git checkout --quiet tags/"${ARVADOS_TAG}" -b "${ARVADOS_TAG}" )
fi

if [ "x${VAGRANT}" = "xyes" ]; then
  EXTRA_STATES_DIR="/home/vagrant/${CONFIG_DIR}/states"
  SOURCE_PILLARS_DIR="/home/vagrant/${CONFIG_DIR}/pillars"
  SOURCE_TESTS_DIR="/home/vagrant/${TESTS_DIR}"
else
  EXTRA_STATES_DIR="${SCRIPT_DIR}/${CONFIG_DIR}/states"
  SOURCE_PILLARS_DIR="${SCRIPT_DIR}/${CONFIG_DIR}/pillars"
  SOURCE_TESTS_DIR="${SCRIPT_DIR}/${TESTS_DIR}"
fi

SOURCE_STATES_DIR="${EXTRA_STATES_DIR}"

echo "Writing pillars and states"

# Replace variables (cluster,  domain, etc) in the pillars, states and tests
# to ease deployment for newcomers
if [ ! -d "${SOURCE_PILLARS_DIR}" ]; then
  echo "${SOURCE_PILLARS_DIR} does not exist or is not a directory. Exiting."
  exit 1
fi
for f in $(ls "${SOURCE_PILLARS_DIR}"/*); do
  sed "s#__ANONYMOUS_USER_TOKEN__#${ANONYMOUS_USER_TOKEN}#g;
       s#__BLOB_SIGNING_KEY__#${BLOB_SIGNING_KEY}#g;
       s#__CONTROLLER_EXT_SSL_PORT__#${CONTROLLER_EXT_SSL_PORT}#g;
       s#__CLUSTER__#${CLUSTER}#g;
       s#__DOMAIN__#${DOMAIN}#g;
       s#__HOSTNAME_EXT__#${HOSTNAME_EXT}#g;
       s#__HOSTNAME_INT__#${HOSTNAME_INT}#g;
       s#__INITIAL_USER_EMAIL__#${INITIAL_USER_EMAIL}#g;
       s#__INITIAL_USER_PASSWORD__#${INITIAL_USER_PASSWORD}#g;
       s#__INITIAL_USER__#${INITIAL_USER}#g;
       s#__LE_AWS_REGION__#${LE_AWS_REGION}#g;
       s#__LE_AWS_SECRET_ACCESS_KEY__#${LE_AWS_SECRET_ACCESS_KEY}#g;
       s#__LE_AWS_ACCESS_KEY_ID__#${LE_AWS_ACCESS_KEY_ID}#g;
       s#__DATABASE_PASSWORD__#${DATABASE_PASSWORD}#g;
       s#__KEEPWEB_EXT_SSL_PORT__#${KEEPWEB_EXT_SSL_PORT}#g;
       s#__KEEP_EXT_SSL_PORT__#${KEEP_EXT_SSL_PORT}#g;
       s#__MANAGEMENT_TOKEN__#${MANAGEMENT_TOKEN}#g;
       s#__RELEASE__#${RELEASE}#g;
       s#__SYSTEM_ROOT_TOKEN__#${SYSTEM_ROOT_TOKEN}#g;
       s#__VERSION__#${VERSION}#g;
       s#__WEBSHELL_EXT_SSL_PORT__#${WEBSHELL_EXT_SSL_PORT}#g;
       s#__WEBSOCKET_EXT_SSL_PORT__#${WEBSOCKET_EXT_SSL_PORT}#g;
       s#__WORKBENCH1_EXT_SSL_PORT__#${WORKBENCH1_EXT_SSL_PORT}#g;
       s#__WORKBENCH2_EXT_SSL_PORT__#${WORKBENCH2_EXT_SSL_PORT}#g;
       s#__CLUSTER_INT_CIDR__#${CLUSTER_INT_CIDR}#g;
       s#__CONTROLLER_INT_IP__#${CONTROLLER_INT_IP}#g;
       s#__WEBSOCKET_INT_IP__#${WEBSOCKET_INT_IP}#g;
       s#__KEEP_INT_IP__#${KEEP_INT_IP}#g;
       s#__KEEPSTORE0_INT_IP__#${KEEPSTORE0_INT_IP}#g;
       s#__KEEPSTORE1_INT_IP__#${KEEPSTORE1_INT_IP}#g;
       s#__KEEPWEB_INT_IP__#${KEEPWEB_INT_IP}#g;
       s#__WEBSHELL_INT_IP__#${WEBSHELL_INT_IP}#g;
       s#__SHELL_INT_IP__#${SHELL_INT_IP}#g;
       s#__WORKBENCH1_INT_IP__#${WORKBENCH1_INT_IP}#g;
       s#__WORKBENCH2_INT_IP__#${WORKBENCH2_INT_IP}#g;
       s#__DATABASE_INT_IP__#${DATABASE_INT_IP}#g;
       s#__WORKBENCH_SECRET_KEY__#${WORKBENCH_SECRET_KEY}#g" \
  "${f}" > "${P_DIR}"/$(basename "${f}")
done

if [ "x${TEST}" = "xyes" ] && [ ! -d "${SOURCE_TESTS_DIR}" ]; then
  echo "You requested to run tests, but ${SOURCE_TESTS_DIR} does not exist or is not a directory. Exiting."
  exit 1
fi
mkdir -p ${T_DIR}
# Replace cluster and domain name in the test files
for f in $(ls "${SOURCE_TESTS_DIR}"/*); do
  sed "s#__CLUSTER__#${CLUSTER}#g;
       s#__CONTROLLER_EXT_SSL_PORT__#${CONTROLLER_EXT_SSL_PORT}#g;
       s#__DOMAIN__#${DOMAIN}#g;
       s#__HOSTNAME_INT__#${HOSTNAME_INT}#g;
       s#__INITIAL_USER_EMAIL__#${INITIAL_USER_EMAIL}#g;
       s#__INITIAL_USER_PASSWORD__#${INITIAL_USER_PASSWORD}#g
       s#__INITIAL_USER__#${INITIAL_USER}#g;
       s#__DATABASE_PASSWORD__#${DATABASE_PASSWORD}#g;
       s#__SYSTEM_ROOT_TOKEN__#${SYSTEM_ROOT_TOKEN}#g" \
  "${f}" > ${T_DIR}/$(basename "${f}")
done
chmod 755 ${T_DIR}/run-test.sh

# Replace helper state files that differ from the formula's examples
if [ -d "${SOURCE_STATES_DIR}" ]; then
  mkdir -p "${F_DIR}"/extra/extra

  for f in $(ls "${SOURCE_STATES_DIR}"/*); do
    sed "s#__ANONYMOUS_USER_TOKEN__#${ANONYMOUS_USER_TOKEN}#g;
         s#__CLUSTER__#${CLUSTER}#g;
         s#__BLOB_SIGNING_KEY__#${BLOB_SIGNING_KEY}#g;
         s#__CONTROLLER_EXT_SSL_PORT__#${CONTROLLER_EXT_SSL_PORT}#g;
         s#__DOMAIN__#${DOMAIN}#g;
         s#__HOSTNAME_EXT__#${HOSTNAME_EXT}#g;
         s#__HOSTNAME_INT__#${HOSTNAME_INT}#g;
         s#__INITIAL_USER_EMAIL__#${INITIAL_USER_EMAIL}#g;
         s#__INITIAL_USER_PASSWORD__#${INITIAL_USER_PASSWORD}#g;
         s#__INITIAL_USER__#${INITIAL_USER}#g;
         s#__DATABASE_PASSWORD__#${DATABASE_PASSWORD}#g;
         s#__KEEPWEB_EXT_SSL_PORT__#${KEEPWEB_EXT_SSL_PORT}#g;
         s#__KEEP_EXT_SSL_PORT__#${KEEP_EXT_SSL_PORT}#g;
         s#__MANAGEMENT_TOKEN__#${MANAGEMENT_TOKEN}#g;
         s#__RELEASE__#${RELEASE}#g;
         s#__SYSTEM_ROOT_TOKEN__#${SYSTEM_ROOT_TOKEN}#g;
         s#__VERSION__#${VERSION}#g;
         s#__CLUSTER_INT_CIDR__#${CLUSTER_INT_CIDR}#g;
         s#__CONTROLLER_INT_IP__#${CONTROLLER_INT_IP}#g;
         s#__WEBSOCKET_INT_IP__#${WEBSOCKET_INT_IP}#g;
         s#__KEEP_INT_IP__#${KEEP_INT_IP}#g;
         s#__KEEPSTORE0_INT_IP__#${KEEPSTORE0_INT_IP}#g;
         s#__KEEPSTORE1_INT_IP__#${KEEPSTORE1_INT_IP}#g;
         s#__KEEPWEB_INT_IP__#${KEEPWEB_INT_IP}#g;
         s#__WEBSHELL_INT_IP__#${WEBSHELL_INT_IP}#g;
         s#__WORKBENCH1_INT_IP__#${WORKBENCH1_INT_IP}#g;
         s#__WORKBENCH2_INT_IP__#${WORKBENCH2_INT_IP}#g;
         s#__DATABASE_INT_IP__#${DATABASE_INT_IP}#g;
         s#__WEBSHELL_EXT_SSL_PORT__#${WEBSHELL_EXT_SSL_PORT}#g;
         s#__WEBSOCKET_EXT_SSL_PORT__#${WEBSOCKET_EXT_SSL_PORT}#g;
         s#__WORKBENCH1_EXT_SSL_PORT__#${WORKBENCH1_EXT_SSL_PORT}#g;
         s#__WORKBENCH2_EXT_SSL_PORT__#${WORKBENCH2_EXT_SSL_PORT}#g;
         s#__WORKBENCH_SECRET_KEY__#${WORKBENCH_SECRET_KEY}#g" \
    "${f}" > "${F_DIR}/extra/extra"/$(basename "${f}")
  done
fi

# Now, we build the SALT states/pillars trees
# As we need to separate both states and pillars in case we want specific
# roles, we iterate on both at the same time

# States
cat > ${S_DIR}/top.sls << EOFTSLS
base:
  '*':
    - locale
EOFTSLS

# Pillars
cat > ${P_DIR}/top.sls << EOFPSLS
base:
  '*':
    - locale
    - arvados
EOFPSLS

# States, extra states
if [ -d "${F_DIR}"/extra/extra ]; then
  if [ "$DEV_MODE" = "yes" ]; then
    # In dev mode, we create some snake oil certs that we'll
    # use as CUSTOM_CERTS, so we don't skip the states file
    SKIP_SNAKE_OIL="dont_snakeoil_certs"
  else
    SKIP_SNAKE_OIL="snakeoil_certs"
  fi
  for f in $(ls "${F_DIR}"/extra/extra/*.sls | grep -v ${SKIP_SNAKE_OIL}); do
  echo "    - extra.$(basename ${f} | sed 's/.sls$//g')" >> ${S_DIR}/top.sls
  done
  # Use custom certs
  if [ "x${USE_LETSENCRYPT}" != "xyes" ]; then
    mkdir -p "${F_DIR}"/extra/extra/files
  fi
fi

# If we want specific roles for a node, just add the desired states
# and its dependencies
if [ -z "${ROLES}" ]; then
  # States
  echo "    - nginx.passenger" >> ${S_DIR}/top.sls
  # Currently, only available on config_examples/multi_host/aws
  if [ "x${USE_LETSENCRYPT}" = "xyes" ]; then
    if [ "x${USE_LETSENCRYPT_IAM_USER}" != "xyes" ]; then
      grep -q "aws_credentials" ${S_DIR}/top.sls || echo "    - extra.aws_credentials" >> ${S_DIR}/top.sls
    fi
    grep -q "letsencrypt"     ${S_DIR}/top.sls || echo "    - letsencrypt" >> ${S_DIR}/top.sls
  else
    # Use custom certs
    # Copy certs to formula extra/files
    # In dev mode, the files will be created and put in the destination directory by the
    # snakeoil_certs.sls state file
    mkdir -p /srv/salt/certs
    cp -rv ${CUSTOM_CERTS_DIR}/* /srv/salt/certs/
    # We add the custom_certs state
    grep -q "custom_certs"    ${S_DIR}/top.sls || echo "    - extra.custom_certs" >> ${S_DIR}/top.sls
  fi

  echo "    - postgres" >> ${S_DIR}/top.sls
  echo "    - docker.software" >> ${S_DIR}/top.sls
  echo "    - arvados" >> ${S_DIR}/top.sls

  # Pillars
  echo "    - docker" >> ${P_DIR}/top.sls
  echo "    - nginx_api_configuration" >> ${P_DIR}/top.sls
  echo "    - nginx_controller_configuration" >> ${P_DIR}/top.sls
  echo "    - nginx_keepproxy_configuration" >> ${P_DIR}/top.sls
  echo "    - nginx_keepweb_configuration" >> ${P_DIR}/top.sls
  echo "    - nginx_passenger" >> ${P_DIR}/top.sls
  echo "    - nginx_websocket_configuration" >> ${P_DIR}/top.sls
  echo "    - nginx_webshell_configuration" >> ${P_DIR}/top.sls
  echo "    - nginx_workbench2_configuration" >> ${P_DIR}/top.sls
  echo "    - nginx_workbench_configuration" >> ${P_DIR}/top.sls
  echo "    - postgresql" >> ${P_DIR}/top.sls

  # Currently, only available on config_examples/multi_host/aws
  if [ "x${USE_LETSENCRYPT}" = "xyes" ]; then
    if [ "x${USE_LETSENCRYPT_IAM_USER}" != "xyes" ]; then
      grep -q "aws_credentials" ${P_DIR}/top.sls || echo "    - aws_credentials" >> ${P_DIR}/top.sls
    fi
    grep -q "letsencrypt"     ${P_DIR}/top.sls || echo "    - letsencrypt" >> ${P_DIR}/top.sls

    # As the pillar differ whether we use LE or custom certs, we need to do a final edition on them
    for c in controller websocket workbench workbench2 webshell download collections keepproxy; do
      sed -i "s/__CERT_REQUIRES__/cmd: create-initial-cert-${c}.${CLUSTER}.${DOMAIN}*/g;
              s#__CERT_PEM__#/etc/letsencrypt/live/${c}.${CLUSTER}.${DOMAIN}/fullchain.pem#g;
              s#__CERT_KEY__#/etc/letsencrypt/live/${c}.${CLUSTER}.${DOMAIN}/privkey.pem#g" \
      ${P_DIR}/nginx_${c}_configuration.sls
    done
  else
    # Use custom certs (either dev mode or prod)
    grep -q "extra_custom_certs" ${P_DIR}/top.sls || echo "    - extra_custom_certs" >> ${P_DIR}/top.sls
    # And add the certs in the custom_certs pillar
    echo "extra_custom_certs_dir: /srv/salt/certs" > ${P_DIR}/extra_custom_certs.sls
    echo "extra_custom_certs:" >> ${P_DIR}/extra_custom_certs.sls

    for c in controller websocket workbench workbench2 webshell download collections keepproxy; do
      grep -q ${c} ${P_DIR}/extra_custom_certs.sls || echo "  - ${c}" >> ${P_DIR}/extra_custom_certs.sls

      # As the pillar differ whether we use LE or custom certs, we need to do a final edition on them
      sed -i "s/__CERT_REQUIRES__/file: extra_custom_certs_file_copy_arvados-${c}.pem/g;
              s#__CERT_PEM__#/etc/nginx/ssl/arvados-${c}.pem#g;
              s#__CERT_KEY__#/etc/nginx/ssl/arvados-${c}.key#g" \
      ${P_DIR}/nginx_${c}_configuration.sls
    done
  fi
else
  # If we add individual roles, make sure we add the repo first
  echo "    - arvados.repo" >> ${S_DIR}/top.sls
  for R in ${ROLES}; do
    case "${R}" in
      "database")
        # States
        echo "    - postgres" >> ${S_DIR}/top.sls
        # Pillars
        echo '    - postgresql' >> ${P_DIR}/top.sls
      ;;
      "api")
        # States
        # FIXME: https://dev.arvados.org/issues/17352
        grep -q "postgres.client" ${S_DIR}/top.sls || echo "    - postgres.client" >> ${S_DIR}/top.sls
        grep -q "nginx.passenger" ${S_DIR}/top.sls || echo "    - nginx.passenger" >> ${S_DIR}/top.sls
        ### If we don't install and run LE before arvados-api-server, it fails and breaks everything
        ### after it. So we add this here as we are, after all, sharing the host for api and controller
        # Currently, only available on config_examples/multi_host/aws
        if [ "x${USE_LETSENCRYPT}" = "xyes" ]; then
          if [ "x${USE_LETSENCRYPT_IAM_USER}" != "xyes" ]; then
            grep -q "aws_credentials" ${S_DIR}/top.sls || echo "    - aws_credentials" >> ${S_DIR}/top.sls
          fi
          grep -q "letsencrypt" ${S_DIR}/top.sls || echo "    - letsencrypt" >> ${S_DIR}/top.sls
        else
          # Use custom certs
          cp -v ${CUSTOM_CERTS_DIR}/controller.* "${F_DIR}/extra/extra/files/"
          # We add the custom_certs state
          grep -q "custom_certs"    ${S_DIR}/top.sls || echo "    - extra.custom_certs" >> ${S_DIR}/top.sls
        fi
        grep -q "arvados.${R}" ${S_DIR}/top.sls    || echo "    - arvados.${R}" >> ${S_DIR}/top.sls
        # Pillars
        grep -q "aws_credentials" ${P_DIR}/top.sls          || echo "    - aws_credentials" >> ${P_DIR}/top.sls
        grep -q "docker" ${P_DIR}/top.sls                   || echo "    - docker" >> ${P_DIR}/top.sls
        grep -q "postgresql" ${P_DIR}/top.sls               || echo "    - postgresql" >> ${P_DIR}/top.sls
        grep -q "nginx_passenger" ${P_DIR}/top.sls          || echo "    - nginx_passenger" >> ${P_DIR}/top.sls
        grep -q "nginx_${R}_configuration" ${P_DIR}/top.sls || echo "    - nginx_${R}_configuration" >> ${P_DIR}/top.sls
      ;;
      "controller" | "websocket" | "workbench" | "workbench2" | "webshell" | "keepweb" | "keepproxy")
        # States
        grep -q "nginx.passenger" ${S_DIR}/top.sls || echo "    - nginx.passenger" >> ${S_DIR}/top.sls
        # Currently, only available on config_examples/multi_host/aws
        if [ "x${USE_LETSENCRYPT}" = "xyes" ]; then
          if [ "x${USE_LETSENCRYPT_IAM_USER}" != "xyes" ]; then
            grep -q "aws_credentials" ${S_DIR}/top.sls || echo "    - aws_credentials" >> ${S_DIR}/top.sls
          fi
          grep -q "letsencrypt"     ${S_DIR}/top.sls || echo "    - letsencrypt" >> ${S_DIR}/top.sls
        else
          # Use custom certs, special case for keepweb
          if [ ${R} = "keepweb" ]; then
            cp -v ${CUSTOM_CERTS_DIR}/download.* "${F_DIR}/extra/extra/files/"
            cp -v ${CUSTOM_CERTS_DIR}/collections.* "${F_DIR}/extra/extra/files/"
          else
            cp -v ${CUSTOM_CERTS_DIR}/${R}.* "${F_DIR}/extra/extra/files/"
          fi
          # We add the custom_certs state
          grep -q "custom_certs"    ${S_DIR}/top.sls || echo "    - extra.custom_certs" >> ${S_DIR}/top.sls

        fi
        # webshell role is just a nginx vhost, so it has no state
        if [ "${R}" != "webshell" ]; then
          grep -q "arvados.${R}" ${S_DIR}/top.sls || echo "    - arvados.${R}" >> ${S_DIR}/top.sls
        fi
        # Pillars
        grep -q "nginx_passenger" ${P_DIR}/top.sls          || echo "    - nginx_passenger" >> ${P_DIR}/top.sls
        grep -q "nginx_${R}_configuration" ${P_DIR}/top.sls || echo "    - nginx_${R}_configuration" >> ${P_DIR}/top.sls
        # Special case for keepweb
        if [ ${R} = "keepweb" ]; then
          grep -q "nginx_download_configuration" ${P_DIR}/top.sls || echo "    - nginx_download_configuration" >> ${P_DIR}/top.sls
          grep -q "nginx_collections_configuration" ${P_DIR}/top.sls || echo "    - nginx_collections_configuration" >> ${P_DIR}/top.sls
        fi

        # Currently, only available on config_examples/multi_host/aws
        if [ "x${USE_LETSENCRYPT}" = "xyes" ]; then
          if [ "x${USE_LETSENCRYPT_IAM_USER}" != "xyes" ]; then
            grep -q "aws_credentials" ${P_DIR}/top.sls || echo "    - aws_credentials" >> ${P_DIR}/top.sls
          fi
          grep -q "letsencrypt"     ${P_DIR}/top.sls || echo "    - letsencrypt" >> ${P_DIR}/top.sls
          grep -q "letsencrypt_${R}_configuration" ${P_DIR}/top.sls || echo "    - letsencrypt_${R}_configuration" >> ${P_DIR}/top.sls

          # As the pillar differ whether we use LE or custom certs, we need to do a final edition on them
          # Special case for keepweb
          if [ ${R} = "keepweb" ]; then
            for kwsub in download collections; do
              sed -i "s/__CERT_REQUIRES__/cmd: create-initial-cert-${kwsub}.${CLUSTER}.${DOMAIN}*/g;
                      s#__CERT_PEM__#/etc/letsencrypt/live/${kwsub}.${CLUSTER}.${DOMAIN}/fullchain.pem#g;
                      s#__CERT_KEY__#/etc/letsencrypt/live/${kwsub}.${CLUSTER}.${DOMAIN}/privkey.pem#g" \
              ${P_DIR}/nginx_${kwsub}_configuration.sls
            done
          else
            sed -i "s/__CERT_REQUIRES__/cmd: create-initial-cert-${R}.${CLUSTER}.${DOMAIN}*/g;
                    s#__CERT_PEM__#/etc/letsencrypt/live/${R}.${CLUSTER}.${DOMAIN}/fullchain.pem#g;
                    s#__CERT_KEY__#/etc/letsencrypt/live/${R}.${CLUSTER}.${DOMAIN}/privkey.pem#g" \
            ${P_DIR}/nginx_${R}_configuration.sls
          fi
        else
          grep -q ${R} ${P_DIR}/extra_custom_certs.sls || echo "  - ${R}" >> ${P_DIR}/extra_custom_certs.sls

          # As the pillar differ whether we use LE or custom certs, we need to do a final edition on them
          # Special case for keepweb
          if [ ${R} = "keepweb" ]; then
            for kwsub in download collections; do
              sed -i "s/__CERT_REQUIRES__/file: extra_custom_certs_file_copy_arvados-${kwsub}.pem/g;
                      s#__CERT_PEM__#/etc/nginx/ssl/arvados-${kwsub}.pem#g;
                      s#__CERT_KEY__#/etc/nginx/ssl/arvados-${kwsub}.key#g" \
              ${P_DIR}/nginx_${kwsub}_configuration.sls
            done
          else
            sed -i "s/__CERT_REQUIRES__/file: extra_custom_certs_file_copy_arvados-${R}.pem/g;
                    s#__CERT_PEM__#/etc/nginx/ssl/arvados-${R}.pem#g;
                    s#__CERT_KEY__#/etc/nginx/ssl/arvados-${R}.key#g" \
            ${P_DIR}/nginx_${R}_configuration.sls
          fi
        fi
      ;;
      "shell")
        # States
        grep -q "docker" ${S_DIR}/top.sls       || echo "    - docker.software" >> ${S_DIR}/top.sls
        grep -q "arvados.${R}" ${S_DIR}/top.sls || echo "    - arvados.${R}" >> ${S_DIR}/top.sls
        # Pillars
        grep -q "" ${P_DIR}/top.sls                             || echo "    - docker" >> ${P_DIR}/top.sls
      ;;
      "dispatcher")
        # States
        grep -q "docker" ${S_DIR}/top.sls       || echo "    - docker.software" >> ${S_DIR}/top.sls
        grep -q "arvados.${R}" ${S_DIR}/top.sls || echo "    - arvados.${R}" >> ${S_DIR}/top.sls
        # Pillars
        # ATM, no specific pillar needed
      ;;
      "keepstore")
        # States
        grep -q "arvados.${R}" ${S_DIR}/top.sls || echo "    - arvados.${R}" >> ${S_DIR}/top.sls
        # Pillars
        # ATM, no specific pillar needed
      ;;
      *)
        echo "Unknown role ${R}"
        exit 1
      ;;
    esac
  done
fi

if [ "${DUMP_CONFIG}" = "yes" ]; then
  # We won't run the rest of the script because we're just dumping the config
  exit 0
fi

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
if [ "$DEV_MODE" = "yes" ]; then
  echo "Copying the Arvados CA certificate to the installer dir, so you can import it"
  # If running in a vagrant VM, also add default user to docker group
  if [ "x${VAGRANT}" = "xyes" ]; then
    cp /etc/ssl/certs/arvados-snakeoil-ca.pem /vagrant/${CLUSTER}.${DOMAIN}-arvados-snakeoil-ca.pem

    echo "Adding the vagrant user to the docker group"
    usermod -a -G docker vagrant
  else
    cp /etc/ssl/certs/arvados-snakeoil-ca.pem ${SCRIPT_DIR}/${CLUSTER}.${DOMAIN}-arvados-snakeoil-ca.pem
  fi
fi

# Test that the installation finished correctly
if [ "x${TEST}" = "xyes" ]; then
  cd ${T_DIR}
  # If we use RVM, we need to run this with it, or most ruby commands will fail
  RVM_EXEC=""
  if [ -x /usr/local/rvm/bin/rvm-exec ]; then
    RVM_EXEC="/usr/local/rvm/bin/rvm-exec"
  fi
  ${RVM_EXEC} ./run-test.sh
fi
