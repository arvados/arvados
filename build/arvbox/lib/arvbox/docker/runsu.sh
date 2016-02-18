#!/bin/sh

HOSTUID=$(ls -nd /usr/src/arvados | sed 's/ */ /' | cut -d' ' -f4)
HOSTGID=$(ls -nd /usr/src/arvados | sed 's/ */ /' | cut -d' ' -f5)

flock /var/lib/arvados/createusers.lock /usr/local/lib/arvbox/createusers.sh

export HOME=/var/lib/arvados

if test -z "$1" ; then
    exec chpst -u arvbox:arvbox:docker $0-service
else
    exec chpst -u arvbox:arvbox:docker $@
fi
