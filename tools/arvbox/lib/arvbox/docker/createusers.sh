#!/bin/bash
# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

set -e -o pipefail

export GEM_HOME=/var/lib/arvados/lib/ruby/gems/2.7.0
export ARVADOS_CONTAINER_PATH=/var/lib/arvados-arvbox

if ! grep "^arvbox:" /etc/passwd >/dev/null 2>/dev/null ; then
    HOSTUID=$(ls -nd /usr/src/arvados | sed 's/ */ /' | cut -d' ' -f4)
    HOSTGID=$(ls -nd /usr/src/arvados | sed 's/ */ /' | cut -d' ' -f5)

    mkdir -p $ARVADOS_CONTAINER_PATH/git $GEM_HOME \
          /var/lib/passenger /var/lib/gopath \
          /var/lib/pip /var/lib/npm

    if test -z "$ARVBOX_HOME" ; then
        ARVBOX_HOME=$ARVADOS_CONTAINER_PATH
    fi

    groupadd --gid $HOSTGID --non-unique arvbox
    groupadd --gid $HOSTGID --non-unique git
    useradd --home-dir $ARVBOX_HOME \
            --uid $HOSTUID --gid $HOSTGID \
            --non-unique \
            --groups docker \
            --shell /bin/bash \
            arvbox
    useradd --home-dir $ARVADOS_CONTAINER_PATH/git --uid $HOSTUID --gid $HOSTGID --non-unique git
    useradd --groups docker crunch

    if [[ "$1" != --no-chown ]] ; then
        chown arvbox:arvbox -R /usr/local $ARVADOS_CONTAINER_PATH $GEM_HOME \
              /var/lib/passenger /var/lib/postgresql \
              /var/lib/nginx /var/log/nginx /etc/ssl/private \
              /var/lib/gopath /var/lib/pip /var/lib/npm \
              /var/lib/arvados
    fi

    mkdir -p /tmp/crunch0 /tmp/crunch1
    chown crunch:crunch -R /tmp/crunch0 /tmp/crunch1

    # singularity needs to be owned by root and suid
    chown root /var/lib/arvados/bin/singularity \
	  /var/lib/arvados/etc/singularity/singularity.conf \
	  /var/lib/arvados/etc/singularity/capability.json \
	  /var/lib/arvados/etc/singularity/ecl.toml
    chmod u+s /var/lib/arvados/bin/singularity

    echo "arvbox    ALL=(crunch) NOPASSWD: ALL" >> /etc/sudoers

    cat <<EOF > /etc/profile.d/paths.sh
export PATH=/var/lib/arvados/bin:/usr/local/bin:/usr/bin:/bin
export GEM_HOME=/var/lib/arvados/lib/ruby/gems/2.7.0
export npm_config_cache=/var/lib/npm
export npm_config_cache_min=Infinity
export R_LIBS=/var/lib/Rlibs
export GOPATH=/var/lib/gopath
EOF

    mkdir -p /etc/arvados
    chown -R arvbox:arvbox /etc/arvados
fi

if ! grep "^fuse:" /etc/group >/dev/null 2>/dev/null ; then
    if test -c /dev/fuse ; then
        FUSEGID=$(ls -nd /dev/fuse | sed 's/ */ /' | cut -d' ' -f5)
        groupadd --gid $FUSEGID --non-unique fuse
        adduser arvbox fuse
        adduser crunch fuse
    fi
fi
