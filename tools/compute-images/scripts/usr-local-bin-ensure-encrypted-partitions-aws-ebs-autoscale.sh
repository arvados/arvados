#!/bin/bash

# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: Apache-2.0

set -e
set -x

MOUNTPATH=/tmp

findmntq() {
    findmnt "$@" >/dev/null
}

ensure_umount() {
    if findmntq "$1"; then
        umount "$1"
    fi
}

# First make sure docker is not using /tmp, then unmount everything under it.
if [ -d /etc/sv/docker.io ]
then
  # TODO: Actually detect Docker state with runit
  DOCKER_ACTIVE=true
  sv stop docker.io || service stop docker.io || true
else
  if systemctl --quiet is-active docker.service docker.socket; then
    systemctl stop docker.service docker.socket || true
    DOCKER_ACTIVE=true
  else
    DOCKER_ACTIVE=false
  fi
fi

ensure_umount "$MOUNTPATH/docker/aufs"

/bin/bash /opt/amazon-ebs-autoscale/install.sh --imdsv2 -f lvm.ext4 -m $MOUNTPATH 2>&1 > /var/log/ebs-autoscale-install.log

# Make sure docker uses the big partition
cat <<EOF > /etc/docker/daemon.json
{
    "data-root": "$MOUNTPATH/docker-data"
}
EOF

if ! $DOCKER_ACTIVE; then
  # Nothing else to do
  exit 0
fi

# restart docker
if [ -d /etc/sv/docker.io ]
then
  ## runit
  sv up docker.io
else
  systemctl start docker.service docker.socket || true
fi

end=$((SECONDS+60))

while [ $SECONDS -lt $end ]; do
  if /usr/bin/docker ps -q >/dev/null; then
    exit 0
  fi
  sleep 1
done

# Docker didn't start within a minute, abort
exit 1
