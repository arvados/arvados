#!/bin/sh

# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: CC-BY-SA-3.0

hostname ${hostname}
echo ${hostname} > /etc/hostname

# Retry just in case internet access is not yet ready
while true; do
  apt-get -o Acquire::ForceIPv4=true update
  ERR=$?
  if [ "$${ERR}" = "0" ]; then
    break
  fi
done

apt-get -o Acquire::ForceIPv4=true install -y git curl

SSH_DIR="/home/${deploy_user}/.ssh"
if [ ! -d "$${SSH_DIR}" ]; then
  install -d -o ${deploy_user} -g ${deploy_user} -m 700 $${SSH_DIR}
fi
echo "${ssh_pubkey}" | install -o ${deploy_user} -g ${deploy_user} -m 600 /dev/stdin $${SSH_DIR}/authorized_keys
