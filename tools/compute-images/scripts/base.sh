#!/bin/bash -euxo pipefail

# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: Apache-2.0

SUDO=sudo

wait_for_apt_locks() {
  while $SUDO fuser /var/{lib/{dpkg,apt/lists},cache/apt/archives}/lock >/dev/null 2>&1; do
    echo "APT: Waiting for apt/dpkg locks to be released..."
    sleep 1
  done
}

# Run apt-get update
$SUDO DEBIAN_FRONTEND=noninteractive apt-get --yes update

# Install gnupg and dirmgr or gpg key checks will fail
wait_for_apt_locks && $SUDO DEBIAN_FRONTEND=noninteractive apt-get -qq --yes install \
  gnupg \
  dirmngr \
  lsb-release

# For good measure, apt-get upgrade
wait_for_apt_locks && $SUDO DEBIAN_FRONTEND=noninteractive apt-get -qq --yes upgrade

# Make sure cloud-init is installed
wait_for_apt_locks && $SUDO DEBIAN_FRONTEND=noninteractive apt-get -qq --yes install cloud-init
if [[ ! -d /var/lib/cloud/scripts/per-boot ]]; then
  mkdir -p /var/lib/cloud/scripts/per-boot
fi

TMP_LSB=`/usr/bin/lsb_release -c -s`
LSB_RELEASE_CODENAME=${TMP_LSB//[$'\t\r\n ']}

# Add the arvados apt repository
echo "# apt.arvados.org" |$SUDO tee --append /etc/apt/sources.list.d/apt.arvados.org.list
echo "deb http://apt.arvados.org/$LSB_RELEASE_CODENAME $LSB_RELEASE_CODENAME${REPOSUFFIX} main" |$SUDO tee --append /etc/apt/sources.list.d/apt.arvados.org.list

# Add the arvados signing key
cat /tmp/1078ECD7.asc | $SUDO apt-key add -
# Add the debian keys
wait_for_apt_locks && $SUDO DEBIAN_FRONTEND=noninteractive apt-get install --yes debian-keyring debian-archive-keyring

# Fix locale
$SUDO /bin/sed -ri 's/# en_US.UTF-8 UTF-8/en_US.UTF-8 UTF-8/' /etc/locale.gen
$SUDO /usr/sbin/locale-gen

# Install some packages we always need
wait_for_apt_locks && $SUDO DEBIAN_FRONTEND=noninteractive apt-get --yes update
wait_for_apt_locks && $SUDO DEBIAN_FRONTEND=noninteractive apt-get -qq --yes install \
  openssh-server \
  apt-utils \
  git \
  curl \
  libcurl3-gnutls \
  libcurl4-openssl-dev \
  lvm2 \
  cryptsetup \
  xfsprogs

# Install the Arvados packages we need
wait_for_apt_locks && $SUDO DEBIAN_FRONTEND=noninteractive apt-get -qq --yes install \
  python3-arvados-fuse \
  crunch-run \
  arvados-docker-cleaner \
  docker.io

# Remove unattended-upgrades if it is installed
wait_for_apt_locks && $SUDO DEBIAN_FRONTEND=noninteractive apt-get -qq --yes remove unattended-upgrades --purge

# Configure arvados-docker-cleaner
$SUDO mkdir -p /etc/arvados/docker-cleaner
$SUDO echo -e "{\n  \"Quota\": \"10G\",\n  \"RemoveStoppedContainers\": \"always\"\n}" > /etc/arvados/docker-cleaner/docker-cleaner.json

# Enable cgroup accounting
$SUDO sed -i 's/GRUB_CMDLINE_LINUX=""/GRUB_CMDLINE_LINUX="cgroup_enable=memory swapaccount=1"/g' /etc/default/grub
$SUDO update-grub

# Set a higher ulimit and the resolver (if set) for docker
if [ "x$RESOLVER" != "x" ]; then
  SET_RESOLVER="--dns ${RESOLVER}"
fi

$SUDO sed "s/ExecStart=\(.*\)/ExecStart=\1 --default-ulimit nofile=10000:10000 ${SET_RESOLVER}/g" \
  /lib/systemd/system/docker.service \
  > /etc/systemd/system/docker.service

$SUDO systemctl daemon-reload

# Make sure user_allow_other is set in fuse.conf
$SUDO sed -i 's/#user_allow_other/user_allow_other/g' /etc/fuse.conf

# Add crunch user with sudo powers
$SUDO adduser --disabled-password --gecos "Crunch user,,,," crunch
# Do not require a password to sudo
echo -e "# for the crunch user\ncrunch ALL=(ALL) NOPASSWD:ALL" | $SUDO tee /etc/sudoers.d/91-crunch

# Set up the ssh public key for the crunch user
$SUDO mkdir /home/crunch/.ssh
$SUDO mv /tmp/crunch-authorized_keys /home/crunch/.ssh/authorized_keys
$SUDO chown -R crunch:crunch /home/crunch/.ssh
$SUDO chmod 600 /home/crunch/.ssh/authorized_keys
$SUDO chmod 700 /home/crunch/.ssh/

# Make sure we resolve via the provided resolver IP if set. Prepending is good enough because
# unless 'rotate' is set, the nameservers are queried in order (cf. man resolv.conf)
if [ "x$RESOLVER" != "x" ]; then
  $SUDO sed -i "s/#prepend domain-name-servers 127.0.0.1;/prepend domain-name-servers ${RESOLVER};/" /etc/dhcp/dhclient.conf
fi
# Set up the cloud-init script that will ensure encrypted disks
$SUDO mv /tmp/usr-local-bin-ensure-encrypted-partitions.sh /usr/local/bin/ensure-encrypted-partitions.sh
$SUDO chmod 755 /usr/local/bin/ensure-encrypted-partitions.sh
$SUDO chown root:root /usr/local/bin/ensure-encrypted-partitions.sh
$SUDO mv /tmp/etc-cloud-cloud.cfg.d-07_compute_arvados_dispatch_cloud.cfg /etc/cloud/cloud.cfg.d/07_compute_arvados_dispatch_cloud.cfg
$SUDO chown root:root /etc/cloud/cloud.cfg.d/07_compute_arvados_dispatch_cloud.cfg
