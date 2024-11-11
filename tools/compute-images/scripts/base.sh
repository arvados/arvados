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

safe_apt() {
    wait_for_apt_locks &&
        $SUDO env DEBIAN_FRONTEND=noninteractive apt-get -q --yes "$@"
}

download_and_install() {
    local url="$1"; shift
    local dest="$1"; shift
    curl -fsSL "$url" | $SUDO install "$@" /dev/stdin "$dest"
}

. /etc/os-release
DISTRO_ID="$ID"
echo "Working directory is '${WORKDIR}'"

### 1. Configure apt preferences

# Third-party packages may depend on contrib packages.
# Make sure we have that component enabled for all existing sources.
if [[ "$DISTRO_ID" = debian ]]; then
    find /etc/apt -name "*.list" -print0 |
        xargs -0r $SUDO sed -ri '/^deb / s/$/ contrib/'
    find /etc/apt -name "*.sources" -print0 |
        xargs -0r $SUDO sed -ri '/^Components:/ s/$/ contrib/'
fi

if [[ "${PIN_PACKAGES:-true}" != false ]]; then
    $SUDO install -d /etc/apt/preferences.d
    $SUDO install -m 0644 \
          "$WORKDIR/etc-apt-preferences.d-arvados.pref" \
          /etc/apt/preferences.d/arvados.pref
fi

### 2. Install all base packages we need

safe_apt update
# Add the debian keys (but don't abort if we can't find them, e.g. on Ubuntu where we don't need them)
safe_apt install debian-keyring debian-archive-keyring 2>/dev/null || true
safe_apt upgrade
# Install gnupg and dirmgr or gpg key checks will fail
safe_apt install \
  gnupg \
  dirmngr \
  lsb-release \
  cloud-init \
  openssh-server \
  apt-utils \
  git \
  curl \
  libcurl3-gnutls \
  libcurl4-openssl-dev \
  lvm2 \
  cryptsetup \
  xfsprogs \
  jq \
  unzip \
  make \
  build-essential \
  libssl-dev \
  uuid-dev \
  squashfs-tools \
  libglib2.0-dev \
  libseccomp-dev

safe_apt remove --purge unattended-upgrades

### 3. Set up third-party apt repositories and install packages we need from them
$SUDO install -d /etc/apt/keyrings

# Add the Arvados apt source
download_and_install https://apt.arvados.org/pubkey.gpg /etc/apt/keyrings/arvados.asc
$SUDO install -m 644 /dev/stdin /etc/apt/sources.list.d/arvados.sources <<EOF
Types: deb
URIs: https://apt.arvados.org/$VERSION_CODENAME
Suites: $VERSION_CODENAME${REPOSUFFIX:-}
Components: main
Signed-by: /etc/apt/keyrings/arvados.asc
EOF

# Add the Docker apt source
DOCKER_URL="https://download.docker.com/linux/$DISTRO_ID"
curl -fsSL "$DOCKER_URL/gpg" | $SUDO gpg --dearmor -o /etc/apt/keyrings/docker.gpg
$SUDO install -m 644 /dev/stdin /etc/apt/sources.list.d/docker.sources <<EOF
Types: deb
URIs: $DOCKER_URL/
Suites: $VERSION_CODENAME
Components: stable
Signed-by: /etc/apt/keyrings/docker.gpg
EOF

# Add the NVIDIA CUDA apt source
# Note that the "keyring" package also installs the apt source
NVIDIA_URL="https://developer.download.nvidia.com/compute/cuda/repos/$(echo "$DISTRO_ID$VERSION_ID" | tr -d .)/x86_64"
CUDA_KEYRING_DEB=cuda-keyring_1.1-1_all.deb
curl -fsSL -o "$WORKDIR/$CUDA_KEYRING_DEB" "$NVIDIA_URL/$CUDA_KEYRING_DEB"
wait_for_apt_locks && $SUDO dpkg -i "$WORKDIR/$CUDA_KEYRING_DEB"

# Add the NVIDIA container toolkit apt source
download_and_install \
    https://nvidia.github.io/libnvidia-container/gpgkey \
    /etc/apt/keyrings/nvidia-container-toolkit.asc
download_and_install \
    https://nvidia.github.io/libnvidia-container/stable/deb/nvidia-container-toolkit.list \
    /etc/apt/sources.list.d/nvidia-container-toolkit.list
$SUDO sed -i 's@^deb http@deb [signed-by=/etc/apt/keyrings/nvidia-container-toolkit.asc] http@' \
      /etc/apt/sources.list.d/nvidia-container-toolkit.list

safe_apt update
safe_apt install python3-arvados-fuse arvados-docker-cleaner
safe_apt install --no-install-recommends docker-ce

### 4. Compute node system configuration

mkdir -p /var/lib/cloud/scripts/per-boot

# Fix locale
$SUDO /bin/sed -ri 's/# en_US.UTF-8 UTF-8/en_US.UTF-8 UTF-8/' /etc/locale.gen
$SUDO /usr/sbin/locale-gen

# Set a higher ulimit and the resolver (if set) for docker
$SUDO install -d /etc/docker
$SUDO install -m 644 /dev/stdin /etc/docker/daemon.json <<EOFDOCKER
{
  "default-ulimits": {
    "nofile": {
      "Hard": 10000,
      "Name": "nofile",
      "Soft": 10000
    }
  }
  ${RESOLVER:+ , \"dns\": \"$RESOLVER\"}
}
EOFDOCKER

# docker should not start on boot: we restart it inside /usr/local/bin/ensure-encrypted-partitions.sh,
# and the BootProbeCommand might be "docker ps -q"
$SUDO systemctl disable docker

# Configure arvados-docker-cleaner
$SUDO mkdir -p /etc/arvados/docker-cleaner
$SUDO echo -e "{\n  \"Quota\": \"10G\",\n  \"RemoveStoppedContainers\": \"always\"\n}" > /etc/arvados/docker-cleaner/docker-cleaner.json

# Enable cgroup accounting (forcing cgroups v1)
$SUDO echo 'GRUB_CMDLINE_LINUX="$GRUB_CMDLINE_LINUX cgroup_enable=memory swapaccount=1 systemd.unified_cgroup_hierarchy=0"' >> /etc/default/grub
$SUDO update-grub

# Make sure user_allow_other is set in fuse.conf
$SUDO sed -i 's/#user_allow_other/user_allow_other/g' /etc/fuse.conf

# Add crunch user with sudo powers
$SUDO adduser --disabled-password --gecos "Crunch user,,,," crunch
# Do not require a password to sudo
echo -e "# for the crunch user\ncrunch ALL=(ALL) NOPASSWD:ALL" | $SUDO tee /etc/sudoers.d/91-crunch
# Set up the ssh public key for the crunch user
$SUDO install -d -m 700 -o crunch -g crunch ~crunch/.ssh
$SUDO install -m 600 -o crunch -g crunch "$WORKDIR/crunch-authorized_keys" ~crunch/.ssh/authorized_keys

# Make sure we resolve via the provided resolver IP if set. Prepending is good enough because
# unless 'rotate' is set, the nameservers are queried in order (cf. man resolv.conf)
if [ -n "${RESOLVER:-}" ]; then
  $SUDO sed -i "s/#prepend domain-name-servers 127.0.0.1;/prepend domain-name-servers ${RESOLVER};/" /etc/dhcp/dhclient.conf
fi

if [ "${AWS_EBS_AUTOSCALE:-}" != "1" ]; then
  # Set up the cloud-init script that will ensure encrypted disks
  $SUDO install "$WORKDIR/usr-local-bin-ensure-encrypted-partitions.sh" /usr/local/bin/ensure-encrypted-partitions.sh
else
  download_and_install "https://awscli.amazonaws.com/awscli-exe-linux-x86_64.zip" "${WORKDIR}/awscliv2.zip"
  unzip -q ${WORKDIR}/awscliv2.zip -d ${WORKDIR} && $SUDO ${WORKDIR}/aws/install
  EBS_AUTOSCALE_VERSION="ee323f0751c2b6f733692e805b51b9bf3c251bac"
  $SUDO env -C /opt git clone https://github.com/arvados/amazon-ebs-autoscale.git
  $SUDO git -C /opt/amazon-ebs-autoscale checkout "$EBS_AUTOSCALE_VERSION"

  # Set up the cloud-init script that makes use of the AWS EBS autoscaler
  $SUDO install "$WORKDIR/usr-local-bin-ensure-encrypted-partitions-aws-ebs-autoscale.sh" /usr/local/bin/ensure-encrypted-partitions.sh
fi

$SUDO install -m 644 \
      "$WORKDIR/etc-cloud-cloud.cfg.d-07_compute_arvados_dispatch_cloud.cfg" \
      /etc/cloud/cloud.cfg.d/07_compute_arvados_dispatch_cloud.cfg

if [ "$NVIDIA_GPU_SUPPORT" == "1" ]; then
  # We need a kernel and matching headers
  if [[ "$DISTRO_ID" == debian ]]; then
    safe_apt install linux-image-cloud-amd64 linux-headers-cloud-amd64
  elif [ "$CLOUD" == "azure" ]; then
    safe_apt install linux-image-azure linux-headers-azure
  elif [ "$CLOUD" == "aws" ]; then
    safe_apt install linux-image-aws linux-headers-aws
  fi
  safe_apt install cuda libnvidia-container1 libnvidia-container-tools nvidia-container-toolkit

  # Various components fail to start, and cause systemd to boot in degraded
  # state, if the system does not actually have an NVIDIA GPU. Configure the
  # image to adapt at boot time.

  # Don't load modules unconditionally.
  # Instead load them if hardware is detected.
  if [[ -f /etc/modules-load.d/nvidia.conf ]]; then
      $SUDO mv /etc/modules-load.d/nvidia.conf /etc/modules-load.d/nvidia.avail
  fi
  $SUDO install "$WORKDIR/usr-local-bin-detect-gpu.sh" /usr/local/bin/detect-gpu.sh
  $SUDO install -d /etc/systemd/system/systemd-modules-load.service.d
  $SUDO install -m 0644 \
        "$WORKDIR/etc-systemd-system-systemd-modules-load.service.d-detect-gpu.conf" \
        /etc/systemd/system/systemd-modules-load.service.d/detect-gpu.conf

  # Don't start the persistence daemon.
  # Instead rely on crunch-run's CUDA initialization.
  if $SUDO systemctl is-enabled --quiet nvidia-persistenced.service; then
    $SUDO systemctl disable nvidia-persistenced.service
  fi
fi

# Get Go and build singularity
mkdir -p /var/lib/arvados
rm -rf /var/lib/arvados/go/
curl -fsSL https://storage.googleapis.com/golang/go${GOVERSION}.linux-amd64.tar.gz |
    tar -C /var/lib/arvados -xz
ln -sf /var/lib/arvados/go/bin/* /usr/local/bin/

singularityversion=3.10.4
cd /var/lib/arvados
git clone --recurse-submodules https://github.com/sylabs/singularity
cd singularity
git checkout v${singularityversion}
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

safe_apt clean
