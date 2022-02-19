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
  sv stop docker.io || service stop docker.io || true
else
  service docker stop || true
fi

ensure_umount "$MOUNTPATH/docker/aufs"

/bin/bash /opt/amazon-ebs-autoscale/install.sh -f lvm.ext4 -m $MOUNTPATH 2>&1 > /var/log/ebs-autoscale-install.log

# Make sure docker uses the big partition
cat <<EOF > /etc/docker/daemon.json
{
    "data-root": "$MOUNTPATH/docker-data"
}
EOF

# restart docker
if [ -d /etc/sv/docker.io ]
then
  ## runit
  sv up docker.io
else
  service docker start
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
