#!/bin/sh
# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: Apache-2.0

apt update
apt install -y curl xfsprogs

# Find the unformatted disks with lsblk, getting those with no format ($2)
# and which name has no number (for xv*) or 'p?' (for nmve*)
UNFORMATTED_DISK="/dev/$(lsblk -o NAME,FSTYPE -dsn | awk '/xv.*[0-9].*/ || /nvme.*p.*/ { next; } $2 == "" {print $1}')"
if ! grep -q '/data' /etc/fstab && [ "$${UNFORMATTED_DISK}" != "/dev/" ]; then
  mkdir -p /data
  mkfs.xfs -f $${UNFORMATTED_DISK} || exit 1
  BLKID=$(blkid |grep xfs|awk '{print $2}')

  echo "# Added by curii_run_once script" >> /etc/fstab
  echo "$${BLKID} /data xfs auto 0 0" >> /etc/fstab
  mount  /data || exit 1
fi
