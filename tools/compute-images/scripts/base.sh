#!/bin/bash -euxo pipefail

# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: Apache-2.0

set -eu -o pipefail

SUDO=sudo

wait_for_apt_locks() {
  while $SUDO fuser /var/{lib/{dpkg,apt/lists},cache/apt/archives}/lock >/dev/null 2>&1; do
    echo "APT: Waiting for apt/dpkg locks to be released..."
    sleep 1
  done
}

# $DIST should not have a dot if there is one in /etc/os-release (e.g. 18.04)
DIST=$(. /etc/os-release; echo $ID$VERSION_ID | tr -d '.')

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

SET_RESOLVER=
if [ -n "$RESOLVER" ]; then
  SET_RESOLVER="--dns ${RESOLVER}"
fi

# Add the arvados apt repository
echo "# apt.arvados.org" |$SUDO tee --append /etc/apt/sources.list.d/apt.arvados.org.list
echo "deb http://apt.arvados.org/$LSB_RELEASE_CODENAME $LSB_RELEASE_CODENAME${REPOSUFFIX} main" |$SUDO tee --append /etc/apt/sources.list.d/apt.arvados.org.list

# Add the arvados signing key
cat /tmp/1078ECD7.asc | $SUDO apt-key add -
# Add the debian keys (but don't abort if we can't find them, e.g. on Ubuntu where we don't need them)
wait_for_apt_locks && $SUDO DEBIAN_FRONTEND=noninteractive apt-get install --yes debian-keyring debian-archive-keyring 2>/dev/null || true

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
  xfsprogs \
  squashfs-tools

# Install the Arvados packages we need
wait_for_apt_locks && $SUDO DEBIAN_FRONTEND=noninteractive apt-get -qq --yes install \
  python3-arvados-fuse \
  arvados-docker-cleaner

# We want Docker 20.10 or later so that we support glibc 2.33 and up in the container, cf.
# https://bugs.debian.org/cgi-bin/bugreport.cgi?bug=1005906
dockerversion=5:20.10.13~3-0
if [[ "$DIST" =~ ^debian ]]; then
  family="debian"
  if [ "$DIST" == "debian10" ]; then
    distro="buster"
  elif [ "$DIST" == "debian11" ]; then
    distro="bullseye"
  fi
elif [[ "$DIST" =~ ^ubuntu ]]; then
  family="ubuntu"
  if [ "$DIST" == "ubuntu1804" ]; then
    distro="bionic"
  elif [ "$DIST" == "ubuntu2004" ]; then
    distro="focal"
  fi
else
  echo "Unsupported distribution $DIST"
  exit 1
fi
curl -fsSL https://download.docker.com/linux/$family/gpg | $SUDO gpg --dearmor -o /usr/share/keyrings/docker-archive-keyring.gpg
echo deb [arch=amd64 signed-by=/usr/share/keyrings/docker-archive-keyring.gpg] https://download.docker.com/linux/$family/ $distro stable | \
    $SUDO tee /etc/apt/sources.list.d/docker.list
$SUDO apt-get update
$SUDO apt-get -yq --no-install-recommends install docker-ce=${dockerversion}~${family}-${distro}

# Set a higher ulimit and the resolver (if set) for docker
$SUDO sed "s/ExecStart=\(.*\)/ExecStart=\1 --default-ulimit nofile=10000:10000 ${SET_RESOLVER}/g" \
  /lib/systemd/system/docker.service \
  > /etc/systemd/system/docker.service

$SUDO systemctl daemon-reload

# docker should not start on boot: we restart it inside /usr/local/bin/ensure-encrypted-partitions.sh,
# and the BootProbeCommand might be "docker ps -q"
$SUDO systemctl disable docker

# Get Go and build singularity
mkdir -p /var/lib/arvados
rm -rf /var/lib/arvados/go/
curl -s https://storage.googleapis.com/golang/go${GOVERSION}.linux-amd64.tar.gz | tar -C /var/lib/arvados -xzf -
ln -sf /var/lib/arvados/go/bin/* /usr/local/bin/

singularityversion=3.9.9
curl -Ls https://github.com/sylabs/singularity/archive/refs/tags/v${singularityversion}.tar.gz | tar -C /var/lib/arvados -xzf -
cd /var/lib/arvados/singularity-${singularityversion}

# build dependencies for singularity
wait_for_apt_locks && $SUDO DEBIAN_FRONTEND=noninteractive apt-get -qq --yes install \
  make build-essential libssl-dev uuid-dev cryptsetup

echo $singularityversion > VERSION
./mconfig --prefix=/var/lib/arvados
make -C ./builddir
make -C ./builddir install
ln -sf /var/lib/arvados/bin/* /usr/local/bin/

# set `mksquashfs mem` in the singularity config file if it is configured
if [ "$MKSQUASHFS_MEM" != "" ]; then
  echo "mksquashfs mem = ${MKSQUASHFS_MEM}" >> /var/lib/arvados/etc/singularity/singularity.conf
fi

# Print singularity version installed
singularity --version

# Remove unattended-upgrades if it is installed
wait_for_apt_locks && $SUDO DEBIAN_FRONTEND=noninteractive apt-get -qq --yes remove unattended-upgrades --purge

# Configure arvados-docker-cleaner
$SUDO mkdir -p /etc/arvados/docker-cleaner
$SUDO echo -e "{\n  \"Quota\": \"10G\",\n  \"RemoveStoppedContainers\": \"always\"\n}" > /etc/arvados/docker-cleaner/docker-cleaner.json

# Enable cgroup accounting
$SUDO sed -i 's/GRUB_CMDLINE_LINUX=""/GRUB_CMDLINE_LINUX="cgroup_enable=memory swapaccount=1"/g' /etc/default/grub
$SUDO update-grub

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

# AWS_EBS_AUTOSCALE is not always set, work around unset variable check
EBS_AUTOSCALE=${AWS_EBS_AUTOSCALE:-}

if [ "$EBS_AUTOSCALE" != "1" ]; then
  # Set up the cloud-init script that will ensure encrypted disks
  $SUDO mv /tmp/usr-local-bin-ensure-encrypted-partitions.sh /usr/local/bin/ensure-encrypted-partitions.sh
else
  wait_for_apt_locks && $SUDO DEBIAN_FRONTEND=noninteractive apt-get -qq --yes install jq unzip

  curl -s "https://awscli.amazonaws.com/awscli-exe-linux-x86_64.zip" -o "/tmp/awscliv2.zip"
  unzip -q /tmp/awscliv2.zip -d /tmp && $SUDO /tmp/aws/install
  # Pinned to v2.4.5 because we apply a patch below
  #export EBS_AUTOSCALE_VERSION=$(curl --silent "https://api.github.com/repos/awslabs/amazon-ebs-autoscale/releases/latest" | jq -r .tag_name)
  export EBS_AUTOSCALE_VERSION="v2.4.5"
  cd /opt && $SUDO git clone https://github.com/awslabs/amazon-ebs-autoscale.git
  cd /opt/amazon-ebs-autoscale && $SUDO git checkout $EBS_AUTOSCALE_VERSION
  $SUDO patch -p1 < /tmp/create-ebs-volume-nvme.patch

  # This script really requires bash and the shebang line is wrong
  $SUDO sed -i 's|^#!/bin/sh|#!/bin/bash|' /opt/amazon-ebs-autoscale/bin/ebs-autoscale

  # Set up the cloud-init script that makes use of the AWS EBS autoscaler
  $SUDO mv /tmp/usr-local-bin-ensure-encrypted-partitions-aws-ebs-autoscale.sh /usr/local/bin/ensure-encrypted-partitions.sh
fi

$SUDO chmod 755 /usr/local/bin/ensure-encrypted-partitions.sh
$SUDO chown root:root /usr/local/bin/ensure-encrypted-partitions.sh
$SUDO mv /tmp/etc-cloud-cloud.cfg.d-07_compute_arvados_dispatch_cloud.cfg /etc/cloud/cloud.cfg.d/07_compute_arvados_dispatch_cloud.cfg
$SUDO chown root:root /etc/cloud/cloud.cfg.d/07_compute_arvados_dispatch_cloud.cfg

if [ "$NVIDIA_GPU_SUPPORT" == "1" ]; then
  # We need a kernel and matching headers
  if [[ "$DIST" =~ ^debian ]]; then
    $SUDO apt-get -y install linux-image-cloud-amd64 linux-headers-cloud-amd64
  elif [ "$CLOUD" == "azure" ]; then
    $SUDO apt-get -y install linux-image-azure linux-headers-azure
  elif [ "$CLOUD" == "aws" ]; then
    $SUDO apt-get -y install linux-image-aws linux-headers-aws
  fi

  # Install CUDA
  $SUDO apt-key adv --fetch-keys https://developer.download.nvidia.com/compute/cuda/repos/$DIST/x86_64/7fa2af80.pub
  $SUDO apt-key adv --fetch-keys https://developer.download.nvidia.com/compute/cuda/repos/$DIST/x86_64/3bf863cc.pub
  $SUDO apt-get -y install software-properties-common
  $SUDO add-apt-repository "deb https://developer.download.nvidia.com/compute/cuda/repos/$DIST/x86_64/ /"
  # Ubuntu 18.04's add-apt-repository does not understand 'contrib'
  $SUDO add-apt-repository contrib || true
  $SUDO apt-get update
  $SUDO apt-get -y install cuda

  # Install libnvidia-container, the tooling for Docker/Singularity
  curl -s -L https://nvidia.github.io/libnvidia-container/gpgkey | \
    $SUDO apt-key add -
  if [ "$DIST" == "debian11" ]; then
    # As of 2021-12-16 libnvidia-container and friends are only available for
    # Debian 10, not yet Debian 11. Install experimental rc1 package as per this
    # workaround:
    # https://github.com/NVIDIA/nvidia-docker/issues/1549#issuecomment-989670662
    curl -s -L https://nvidia.github.io/libnvidia-container/debian10/libnvidia-container.list | \
      $SUDO tee /etc/apt/sources.list.d/libnvidia-container.list
    $SUDO sed -i -e '/experimental/ s/^#//g' /etc/apt/sources.list.d/libnvidia-container.list
  else
    # here, $DIST should have a dot if there is one in /etc/os-release (e.g. 18.04)...
    DIST=$(. /etc/os-release; echo $ID$VERSION_ID)
    curl -s -L https://nvidia.github.io/libnvidia-container/$DIST/libnvidia-container.list | \
      $SUDO tee /etc/apt/sources.list.d/libnvidia-container.list
  fi

  $SUDO apt-get update
  $SUDO apt-get -y install libnvidia-container1 libnvidia-container-tools nvidia-container-toolkit
  # This service fails to start when the image is booted without Nvidia GPUs present, which makes
  # `systemctl is-system-running` respond with "degraded" and since that command is our default
  # BootProbeCommand, compute nodes never finish booting from Arvados' perspective.
  # Disable the service to avoid this. This should be fine because crunch-run does its own basic
  # CUDA initialization.
  $SUDO systemctl disable nvidia-persistenced.service
fi

$SUDO apt-get clean
