#!/bin/bash

# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: CC-BY-SA-3.0

set -e

sync() {
    if test "$NODE" != localhost ; then
	if ! ssh $NODE test -d arvados-setup ; then
	    ssh $NODE git init --bare arvados-setup.git
	    if ! git remote add $NODE $DEPLOY_USER@$NODE:arvados-setup.git ; then
		git remote set-url $NODE $DEPLOY_USER@$NODE:arvados-setup.git
	    fi
	    git push $NODE $BRANCH
	    ssh $NODE git clone arvados-setup.git arvados-setup
	fi

	git push $NODE $BRANCH
	ssh $NODE git -C arvados-setup checkout $BRANCH
	ssh $NODE git -C arvados-setup pull
    fi
}

deploynode() {
    if test -z "${NODES[$NODE]}" ; then
	echo "No roles declared for '$NODE' in local.params"
	exit 1
    fi

    if test $NODE = localhost ; then
	sudo ./provision.sh --config local.params --roles ${NODES[$NODE]}
    else
	ssh $DEPLOY_USER@$NODE "cd arvados-setup && sudo ./provision.sh --config local.params --roles ${NODES[$NODE]}"
    fi
}

subcmd="$1"
if test -n "$subcmd" ; then
    shift
fi
case "$subcmd" in
    initialize)
	if ! test -f provision.sh ; then
	    echo "Must be run from arvados/tools/salt-install"
	    exit
	fi

	SETUPDIR=$1
	PARAMS=$2
	SLS=$3

	err=
	if test -z "$PARAMS" -o ! -f local.params.example.$PARAMS ; then
	    echo "Not found: local.params.example.$PARAMS"
	    echo "Expected one of multiple_hosts, single_host_multiple_hostnames, single_host_single_hostname"
	    err=1
	fi

	if test -z "$SLS" -o ! -d config_examples/$SLS ; then
	    echo "Not found: config_examples/$SLS"
	    echo "Expected one of multi_host/aws, single_host/multiple_hostnames, single_host/single_hostname"
	    err=1
	fi

	if test -z "$SETUPDIR" -o -z "$PARAMS" -o -z "$SLS" ; then
	    echo "installer.sh <setup dir to initialize> <params template> <config template>"
	    err=1
	fi

	if test -n "$err" ; then
	    exit 1
	fi

	echo "Initializing $SETUPDIR"
	git init $SETUPDIR
	cp -r *.sh tests $SETUPDIR

	cp local.params.example.$PARAMS $SETUPDIR/local.params
	cp -r config_examples/$SLS $SETUPDIR/local_config_dir

	cd $SETUPDIR
	git add *.sh local.params local_config_dir tests
	git commit -m"initial commit"

	echo "setup directory initialized, now go to $SETUPDIR, edit 'local.params' and 'local_config_dir' as needed, then run 'installer.sh deploy'"
	;;
    deploy)
	NODE=$1
	CONFIG_FILE=local.params
	if ! test -s $CONFIG_FILE ; then
	    echo "Must be run from arvados-setup, maybe you need to 'initialize' first?"
	fi

	source ${CONFIG_FILE}

	set -x

	BRANCH=$(git branch --show-current)

	git add -A
	if ! git diff --cached --exit-code ; then
	    git commit -m"prepare for deploy"
	fi

	if test -z "$NODE"; then
	    for NODE in "${!NODES[@]}"
	    do
		sync
		deploynode
	    done
	else
	    sync
	    deploynode
	fi
	;;
    *)
	echo "Arvados installer"
	echo ""
	echo "initialize   initialize the setup directory for configuration"
	echo "deploy       deploy the configuration from the setup directory"
	;;
esac
