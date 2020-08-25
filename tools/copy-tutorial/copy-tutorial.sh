#!/bin/sh
# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

set -e

if test -z "$1"  ; then
  echo "$0: Copies Arvados tutorial resources from public data cluster (jutro)"
  echo "Usage: copy-tutorial.sh <dest>"
  echo "<dest> is destination cluster configuration that can be found in ~/.config/arvados"
  exit
fi

echo "Copying from public data cluster (jutro) to $1"

for a in $(cat $HOME/.config/arvados/$1.conf) ; do export $a ; done

project_uuid=$(arv --format=uuid group create --group '{"name":"User guide resources", "group_class": "project"}')

# Bwa-mem workflow
arv-copy --src jutro --dst $1 --project-uuid=$project_uuid f141fc27e7cfa7f7b6d208df5e0ee01b+59
arv-copy --src jutro --dst $1 --project-uuid=$project_uuid jutro-7fd4e-mkmmq53m1ze6apx

echo "Data copied to \"User guide resources\" at $project_uuid"
