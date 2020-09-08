#!/bin/bash
# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

set -e -o pipefail

if test -z "$1" -o -z "$2"  ; then
  echo "$0: Copies Arvados tutorial resources from public data cluster (jutro)"
  echo "Usage: copy-tutorial.sh <dest> <tutorial>"
  echo "<dest> is 5-character cluster id of the destination"
  echo "<tutorial> is which tutorial to copy, one of:"
  echo " bwa-mem        Tutorial from https://doc.arvados.org/user/tutorials/tutorial-workflow-workbench.html"
  echo " whole-genome   Whole genome variant calling tutorial workflow (large)"
  exit
fi

src=jutro
dest=$1
tutorial=$2

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
echo "Copying from public data cluster (jutro) to $dest"
echo

make_project() {
    name="$1"
    owner="$2"
    if test -z "$owner" ; then
	owner=$(arv --format=uuid user current)
    fi
    project_uuid=$(arv --format=uuid group list --filters '[["name", "=", "'"$name"'"], ["owner_uuid", "=", "'$owner'"]]')
    if test -z "$project_uuid" ; then
	project_uuid=$(arv --format=uuid group create --group '{"name":"'"$name"'", "group_class": "project", "owner_uuid": "'$owner'"}')
	link=$(arv link create --link '{"link_class": "permission", "name": "can_read", "tail_uuid": "'$dest'-j7d0g-anonymouspublic", "head_uuid": "'$project_uuid'"}')
    fi
    echo $project_uuid
}

copy_jobs_image() {
    if ! arv-keepdocker | grep "arvados/jobs *latest" ; then
	arv-copy --src jutro --dst $dest --project-uuid=$project_uuid jutro-4zz18-sxmit0qs6i9n2s4
    fi
}

parent_project=$(make_project "Tutorial projects")
copy_jobs_image

if test "$tutorial" = "bwa-mem" ; then
    echo
    echo "Copying bwa mem tutorial"
    echo

    set -x

    project_uuid=$(make_project 'User guide resources' $parent_project)

    # Bwa-mem workflow
    arv-copy --src jutro --dst $dest --project-uuid=$project_uuid jutro-7fd4e-mkmmq53m1ze6apx

    set +x

    echo
    echo "Finished, data copied to \"User guide resources\" at $project_uuid"
    echo "You can now go to Workbench and choose 'Run a process' and then select 'bwa-mem.cwl'"
    echo
fi

if test "$tutorial" = "whole-genome" ; then
    echo
    echo "Copying whole genome variant calling tutorial"
    echo

    set -x

    project_uuid=$(make_project 'WGS Processing Tutorial' $parent_project)

    # WGS workflow
    arv-copy --src jutro --dst $dest --project-uuid=$project_uuid jutro-7fd4e-tnxg9ytblbxm26i

    set +x

    echo
    echo "Finished, data copied to \"WGS Processing Tutorial\" at $project_uuid"
    echo "You can now go to Workbench and choose 'Run a process' and then select 'bwa-mem.cwl'"
    echo
fi
