#!/bin/bash
# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

set -e -o pipefail

if test -z "$1" ; then
  echo "$0: Copies Arvados tutorial resources from public data cluster (jutro)"
  echo "Usage: copy-tutorial.sh <tutorial>"
  echo "<tutorial> is which tutorial to copy, one of:"
  echo " bwa-mem        Tutorial from https://doc.arvados.org/user/tutorials/tutorial-workflow-workbench.html"
  echo " whole-genome   Whole genome variant calling tutorial workflow (large)"
  exit
fi

if test -z "ARVADOS_API_HOST" ; then
    echo "Please set ARVADOS_API_HOST to the destination cluster"
    exit
fi

src=jutro
tutorial=$1

if ! test -f $HOME/.config/arvados/jutro.conf ; then
    # Set it up with the anonymous user token.
    echo "ARVADOS_API_HOST=jutro.arvadosapi.com" > $HOME/.config/arvados/jutro.conf
    echo "ARVADOS_API_TOKEN=v2/jutro-gj3su-e2o9x84aeg7q005/22idg1m3zna4qe4id3n0b9aw86t72jdw8qu1zj45aboh1mm4ej" >> $HOME/.config/arvados/jutro.conf
    exit 1
fi

echo
echo "Copying from public data cluster (jutro) to $ARVADOS_API_HOST"
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

    fi
    echo $project_uuid
}

copy_jobs_image() {
    if ! arv-keepdocker | grep "arvados/jobs *latest" ; then
	arv-copy --project-uuid=$parent_project jutro-4zz18-sxmit0qs6i9n2s4
    fi
}

parent_project=$(make_project "Tutorial projects")
copy_jobs_image

if test "$tutorial" = "bwa-mem" ; then
    echo
    echo "Copying bwa mem tutorial"
    echo

    arv-copy --project-uuid=$parent_project jutro-j7d0g-rehmt1w5v2p2drp

    echo
    echo "Finished, data copied to \"User guide resources\" at $parent_project"
    echo "You can now go to Workbench and choose 'Run a process' and then select 'bwa-mem.cwl'"
    echo
fi

if test "$tutorial" = "whole-genome" ; then
    echo
    echo "Copying whole genome variant calling tutorial"
    echo

    arv-copy --project-uuid=$parent_project jutro-j7d0g-n2g87m02rsl4cx2

    echo
    echo "Finished, data copied to \"WGS Processing Tutorial\" at $parent_project"
    echo "You can now go to Workbench and choose 'Run a process' and then select 'WGS Processing Tutorial'"
    echo
fi
