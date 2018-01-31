#!/bin/bash
# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

exec 2>&1
sleep 2
set -eux -o pipefail

. /usr/local/lib/arvbox/common.sh
. /usr/local/lib/arvbox/go-setup.sh

flock /var/lib/gopath/gopath.lock go get -t "git.curoverse.com/arvados.git/services/keepstore"
install $GOPATH/bin/keepstore /usr/local/bin

if test "$1" = "--only-deps" ; then
    exit
fi

mkdir -p /var/lib/arvados/$1

export ARVADOS_API_HOST=$localip:${services[api]}
export ARVADOS_API_HOST_INSECURE=1
export ARVADOS_API_TOKEN=$(cat /var/lib/arvados/superuser_token)

set +e
read -rd $'\000' keepservice <<EOF
{
 "service_host":"$localip",
 "service_port":$2,
 "service_ssl_flag":false,
 "service_type":"disk"
}
EOF
set -e

if test -s /var/lib/arvados/$1-uuid ; then
    keep_uuid=$(cat /var/lib/arvados/$1-uuid)
    arv keep_service update --uuid $keep_uuid --keep-service "$keepservice"
else
    UUID=$(arv --format=uuid keep_service create --keep-service "$keepservice")
    echo $UUID > /var/lib/arvados/$1-uuid
fi

set +e
killall -HUP keepproxy

exec /usr/local/bin/keepstore \
     -listen=:$2 \
     -enforce-permissions=true \
     -blob-signing-key-file=/var/lib/arvados/blob_signing_key \
     -data-manager-token-file=/var/lib/arvados/superuser_token \
     -max-buffers=20 \
     -volume=/var/lib/arvados/$1
