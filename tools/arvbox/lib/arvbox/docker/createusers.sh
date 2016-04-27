#!/bin/bash

set -e -o pipefail

if ! grep "^arvbox:" /etc/passwd >/dev/null 2>/dev/null ; then
    HOSTUID=$(ls -nd /usr/src/arvados | sed 's/ */ /' | cut -d' ' -f4)
    HOSTGID=$(ls -nd /usr/src/arvados | sed 's/ */ /' | cut -d' ' -f5)
    FUSEGID=$(ls -nd /dev/fuse | sed 's/ */ /' | cut -d' ' -f5)

    mkdir -p /var/lib/arvados/git /var/lib/gems \
          /var/lib/passenger /var/lib/gopath /var/lib/pip

    groupadd --gid $HOSTGID --non-unique arvbox
    groupadd --gid $FUSEGID --non-unique fuse
    groupadd --gid $HOSTGID --non-unique git
    useradd --home-dir /var/lib/arvados \
            --uid $HOSTUID --gid $HOSTGID \
            --non-unique \
            --groups docker,fuse \
            arvbox
    useradd --home-dir /var/lib/arvados/git --uid $HOSTUID --gid $HOSTGID --non-unique git
    useradd --groups docker,fuse crunch

    chown arvbox:arvbox -R /usr/local /var/lib/arvados /var/lib/gems \
          /var/lib/passenger /var/lib/postgresql \
          /var/lib/nginx /var/log/nginx /etc/ssl/private \
          /var/lib/gopath /var/lib/pip

    mkdir -p /var/lib/gems/ruby/2.1.0
    chown arvbox:arvbox -R /var/lib/gems/ruby/2.1.0

    mkdir -p /tmp/crunch0 /tmp/crunch1
    chown crunch:crunch -R /tmp/crunch0 /tmp/crunch1

    echo "arvbox    ALL=(crunch) NOPASSWD: ALL" >> /etc/sudoers
fi
