#!/bin/bash
# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

# If you want to test arvados in a single host, you can run this script, which
# will install it using salt masterless
# This script is run by the Vagrant file when you run it with
#
# vagrant up

# This could have been done with the Salt vagrant provisioner, but this script
# can be used in environments other than vagrant.

# If branch is set, the script will switch to it before running salt
BRANCH="refactor-config-add-service"

CLUSTER="arva2"
DOMAIN="arv.local"

# Salt's dir
## states
S_DIR="/srv/salt"
## formulas
F_DIR="/srv/formulas"
##pillars
P_DIR="/srv/pillars"
# In vagrant, we can use the shared dir
P_DIR="/vagrant/salt_pillars"

sudo apt-get update
sudo apt-get install -y curl git

dpkg -l |grep salt-minion
if [ ${?} -eq 0 ]; then
  echo "Salt already installed"
else
  curl -L https://bootstrap.saltstack.com -o /tmp/bootstrap_salt.sh
  sudo sh /tmp/bootstrap_salt.sh -XUdfP -x python3
  sudo /bin/systemctl disable salt-minion.service
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

cat > ${S_DIR}/top.sls << EOFTSLS
base:
  '*':
    - example_add_snakeoil_certs
    - nginx.passenger
    - postgres
    - docker
    - arvados
EOFTSLS

cat > ${P_DIR}/top.sls << EOFPSLS
base:
  '*':
    - arvados
    - nginx_api_configuration	
    - nginx_controller_configuration
    - nginx_keepproxy_configuration
    - nginx_keepweb_configuration
    - nginx_passenger		
    - nginx_websocket_configuration
    - nginx_workbench2_configuration
    - nginx_workbench_configuration
    - postgresql
EOFPSLS


# Get the formula and dependencies
cd ${F_DIR} || exit 1
for f in postgres arvados nginx docker; do
  git clone https://github.com/netmanagers/${f}-formula.git
done

if [ "x${BRANCH}" != "x" ]; then
  cd ${F_DIR}/arvados-formula
  git checkout -t origin/${BRANCH}
  cd -
fi

sed "s/example.net/${DOMAIN}/g; s/name: fixme/name: ${CLUSTER}/g" \
  ${F_DIR}/arvados-formula/test/salt/pillar/arvados_dev.sls > ${P_DIR}/arvados.sls

# Replace domain name in the example pillars
for f in ${F_DIR}/arvados-formula/test/salt/pillar/examples/*; do
  sed "s/example.net/${DOMAIN}/g" ${f} > ${P_DIR}/$(basename ${f})
done

# # Copy arvados' pillar.example file to the pillars dir, so it's used
# sed "s/example.net/${DOMAIN}/g" ${F_DIR}/arvados-formula/pillar.example > ${P_DIR}/arvados.sls
# 
# # Replace domain name in the example pillars
# for f in ${F_DIR}/arvados-formula/test/salt/pillar/examples/*; do
#   sed "s/example.net/${DOMAIN}/g" ${f} > ${P_DIR}/$(basename ${f})
# done
# 
# Let's write a /etc/hosts file that points all the hosts to localhost

echo "127.0.0.2 api keep keep0 collections download ws workbench workbench2 api.${CLUSTER}.${DOMAIN} keep.${CLUSTER}.${DOMAIN} keep0.${CLUSTER}.${DOMAIN} collections.${CLUSTER}.${DOMAIN} download.${CLUSTER}.${DOMAIN} ws.${CLUSTER}.${DOMAIN} workbench.${CLUSTER}.${DOMAIN} workbench2.${CLUSTER}.${DOMAIN}" >> /etc/hosts

# Now run the install
salt-call --local state.apply -l debug
