#!/bin/bash
# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

set -e -o pipefail

if test -z "$1"  ; then
  echo "$0: Copies Arvados tutorial resources from public data cluster (jutro)"
  echo "Usage: copy-tutorial.sh <dest>"
  echo "<dest> is 5-character cluster id of the destination"
  exit
fi

src=jutro
dest=$1

if ! test -f $HOME/.config/arvados/${dest}.conf ; then
    echo "Please create $HOME/.config/arvados/${dest}.conf with the following contents:"
    echo "ARVADOS_API_HOST=<${dest} host>"
    echo "ARVADOS_API_TOKEN=<${dest} token>"
    exit 1
fi

if ! test -f $HOME/.config/arvados/jutro.conf ; then
    # Set it up with the anonymous user token.
    echo "ARVADOS_API_HOST=jutro.arvadosapi.com" > $HOME/.config/arvados/jutro.conf
    echo "ARVADOS_API_TOKEN=v2/jutro-gj3su-e2o9x84aeg7q005/22idg1m3zna4qe4id3n0b9aw86t72jdw8qu1zj45aboh1mm4ej" >> $HOME/.config/arvados/jutro.conf
    exit 1
fi

for a in $(cat $HOME/.config/arvados/${dest}.conf) ; do export $a ; done

echo
echo "Copying bwa mem example from public data cluster (jutro) to $dest"
echo

set -x

project_uuid=$(arv --format=uuid group list --filters '[["name", "=", "User guide resources"]]')
if test -z "$project_uuid" ; then
    project_uuid=$(arv --format=uuid group create --group '{"name":"User guide resources", "group_class": "project"}')
    arv link create --link '{"link_class": "permission", "name": "can_read", "tail_uuid": "'$dest'-j7d0g-anonymouspublic", "head_uuid": "'$project_uuid'"}'
fi

if ! arv-keepdocker | grep "arvados/jobs *latest" ; then
    arv-copy --src jutro --dst $dest --project-uuid=$project_uuid jutro-4zz18-sxmit0qs6i9n2s4
fi

# Bwa-mem workflow
arv-copy --src jutro --dst $dest --project-uuid=$project_uuid f141fc27e7cfa7f7b6d208df5e0ee01b+59
arv-copy --src jutro --dst $dest --project-uuid=$project_uuid jutro-7fd4e-mkmmq53m1ze6apx

set +x

echo
echo "Finished, data copied to \"User guide resources\" at $project_uuid"
echo
