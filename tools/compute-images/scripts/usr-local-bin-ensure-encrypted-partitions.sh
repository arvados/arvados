#!/bin/bash

# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: Apache-2.0

set -e
set -x

VGNAME=compute
LVNAME=tmp
LVPATH="/dev/mapper/${VGNAME}-${LVNAME}"
CRYPTPATH=/dev/mapper/tmp
MOUNTPATH=/tmp

findmntq() {
    findmnt "$@" >/dev/null
}

ensure_umount() {
    if findmntq "$1"; then
        umount "$1"
    fi
}

if findmntq --source "$CRYPTPATH" --target "$MOUNTPATH"; then
    exit 0
fi

CLOUD_SERVER=""
while [[ ! "$CLOUD_SERVER" ]]; do
    CLOUD_SERVER="$(curl --silent --head http://169.254.169.254/ \
                    | awk '($1 == "Server:"){sub("\\r+$", ""); print substr($0, 9)}')"
done

DISK_PATTERN=""
case "$CLOUD_SERVER" in
    # EC2
    EC2ws) DISK_PATTERN=/dev/xvd ;;
    # GCP
    "Metadata Server for VM") DISK_PATTERN=/dev/sd ;;
    # Azure
    Microsoft-IIS/*) DISK_PATTERN=/dev/sd ;;
esac

if [[ -z "$DISK_PATTERN" ]]; then
    echo "ensure-encrypted-partitions: Unknown disk configuration; can't run." >&2
    exit 3
fi

declare -a LVM_DEVS=()

ROOT_PARTITION=`findmnt / -f -o source -n`
if [[ "$ROOT_PARTITION" =~ ^\/dev\/nvme ]]; then
  # e.g. /dev/nvme0n1p1, strip last 4 characters
  ROOT_DEVICE_STRING=${ROOT_PARTITION%????}
else
  # e.g. /dev/xvda1, strip last character
  ROOT_DEVICE_STRING=${ROOT_PARTITION//[0-9]/}
fi

# Newer AWS node types use another pattern, /dev/nvmeXn1 for fast instance SSD disks
if [[ "$CLOUD_SERVER" == "EC2ws" ]]; then
  for dev in `ls /dev/nvme* 2>/dev/null`; do
    if [[ "$dev" == "$ROOT_PARTITION" ]] || [[ "$dev" =~ ^$ROOT_DEVICE_STRING ]]; then
      continue
    fi
    if [[ -e ${dev}n1 ]]; then
      ensure_umount "${dev}n1"
      if [[ "$devtype" = disk ]]; then
        dd if=/dev/zero of="${dev}n1" bs=512 count=1
      fi
      LVM_DEVS+=("${dev}n1")
    fi
  done
fi

# Look for traditional disks but only if we're not on AWS or if we haven't found
# a fast instance /dev/nvmeXn1 disk
if [[ "$CLOUD_SERVER" != "EC2ws" ]] || [[ ${#LVM_DEVS[@]} -eq 0 ]]; then
  for dev in `ls $DISK_PATTERN* 2>/dev/null`; do
    # On Azure, we are dealing with /dev/sdb1, on GCP, /dev/sdb, on AWS, /dev/xvdb
    if [[ "$dev" == "$ROOT_PARTITION" ]] || [[ "$dev" =~ ^$ROOT_DEVICE_STRING ]]; then
      continue
    fi
    if [[ ! "$dev" =~ [a-z]$ ]]; then
      continue
    fi
    if [[ -e ${dev}1 ]]; then
        dev=${dev}1
        devtype=partition
    else
        devtype=disk
    fi
    ensure_umount "$dev"
    if [[ "$devtype" = disk ]]; then
        dd if=/dev/zero of="$dev" bs=512 count=1
    fi
    LVM_DEVS+=("$dev")
  done
fi

if [[ "${#LVM_DEVS[@]}" -eq 0 ]]; then
    echo "ensure-encrypted-partitions: No extra disks found." >&2
    exit 4
fi

vgcreate --force --yes "$VGNAME" "${LVM_DEVS[@]}"
lvcreate --extents 100%FREE --name "$LVNAME" "$VGNAME"

KEYPATH="$(mktemp -p /var/tmp key-XXXXXXXX.tmp)"
modprobe dm_mod aes sha256
head -c321 /dev/urandom >"$KEYPATH"
echo YES | cryptsetup luksFormat "$LVPATH" "$KEYPATH"
cryptsetup --key-file "$KEYPATH" luksOpen "$LVPATH" "$(basename "$CRYPTPATH")"
shred -u "$KEYPATH"
mkfs.xfs -f "$CRYPTPATH"

# First make sure docker is not using /tmp, then unmount everything under it.
if [ -d /etc/sv/docker.io ]
then
  sv stop docker.io || service stop docker.io || true
else
  systemctl disable --now docker.service docker.socket || true
fi

ensure_umount "$MOUNTPATH/docker/aufs"

MOUNTOPTIONS="async"
mount -o ${MOUNTOPTIONS} "$CRYPTPATH" "$MOUNTPATH"
chmod a+w,+t "$MOUNTPATH"

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
  systemctl enable --now docker.service docker.socket || true
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
