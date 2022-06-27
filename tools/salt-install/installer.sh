#!/bin/bash

# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: CC-BY-SA-3.0

set -e

declare -A NODES

sync() {
    if [[ "$NODE" != localhost ]] ; then
	if ! ssh $NODE test -d ${GITTARGET}.git ; then
	    ssh $NODE git init --bare ${GITTARGET}.git
	    if ! git remote add $NODE $DEPLOY_USER@$NODE:${GITTARGET}.git ; then
		git remote set-url $NODE $DEPLOY_USER@$NODE:${GITTARGET}.git
	    fi
	    git push $NODE $BRANCH
	    ssh $NODE git clone ${GITTARGET}.git ${GITTARGET}
	fi

	git push $NODE $BRANCH
	ssh $NODE git -C ${GITTARGET} checkout $BRANCH
	ssh $NODE git -C ${GITTARGET} pull
    fi
}

deploynode() {
    if [[ -z "${NODES[$NODE]}" ]] ; then
	echo "No roles declared for '$NODE' in local.params"
	exit 1
    fi

    if [[ "$NODE" = localhost ]] ; then
	sudo ./provision.sh --config local.params --roles ${NODES[$NODE]}
    else
	ssh $DEPLOY_USER@$NODE "cd ${GITTARGET} && sudo ./provision.sh --config local.params --roles ${NODES[$NODE]}"
    fi
}

loadconfig() {
    CONFIG_FILE=local.params
    if [[ ! -s $CONFIG_FILE ]] ; then
	echo "Must be run from initialized setup dir, maybe you need to 'initialize' first?"
    fi
    source ${CONFIG_FILE}
    GITTARGET=arvados-deploy-config-${CLUSTER}
}

subcmd="$1"
if [[ -n "$subcmd" ]] ; then
    shift
fi
case "$subcmd" in
    initialize)
	if [[ ! -f provision.sh ]] ; then
	    echo "Must be run from arvados/tools/salt-install"
	    exit
	fi

	SETUPDIR=$1
	PARAMS=$2
	SLS=$3

	err=
	if [[ -z "$PARAMS" || ! -f local.params.example.$PARAMS ]] ; then
	    echo "Not found: local.params.example.$PARAMS"
	    echo "Expected one of multiple_hosts, single_host_multiple_hostnames, single_host_single_hostname"
	    err=1
	fi

	if [[ -z "$SLS" || ! -d config_examples/$SLS ]] ; then
	    echo "Not found: config_examples/$SLS"
	    echo "Expected one of multi_host/aws, single_host/multiple_hostnames, single_host/single_hostname"
	    err=1
	fi

	if [[ -z "$SETUPDIR" || -z "$PARAMS" || -z "$SLS" ]]; then
	    echo "installer.sh <setup dir to initialize> <params template> <config template>"
	    err=1
	fi

	if [[ -n "$err" ]] ; then
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

	loadconfig

	set -x

	BRANCH=$(git branch --show-current)

	git add -A
	if ! git diff --cached --exit-code ; then
	    git commit -m"prepare for deploy"
	fi

	if [[ -z "$NODE" ]]; then
	    for NODE in "${!NODES[@]}"
	    do
		# Do 'database' role first,
		if [[ "${NODES[$NODE]}" =~ database ]] ; then
		    sync
		    deploynode
		    unset NODES[$NODE]
		fi
	    done

	    for NODE in "${!NODES[@]}"
	    do
		# then  'api' or 'controller' roles
		if [[ "${NODES[$NODE]}" =~ (api|controller) ]] ; then
		    sync
		    deploynode
		    unset NODES[$NODE]
		fi
	    done

	    for NODE in "${!NODES[@]}"
	    do
		# Everything else
		sync
		deploynode
	    done
	else
	    sync
	    deploynode
	fi

	;;
    diagnostics)
	loadconfig

	if ! which arvados-client ; then
	    apt-get install arvados-client
	fi

	export ARVADOS_API_HOST="${CLUSTER}.${DOMAIN}"
	export ARVADOS_API_TOKEN="$SYSTEM_ROOT_TOKEN"

	arvados-client diagnostics -internal-client
	;;
    *)
	echo "Arvados installer"
	echo ""
	echo "initialize   initialize the setup directory for configuration"
	echo "deploy       deploy the configuration from the setup directory"
	echo "diagnostics  check your install using diagnostics"
	;;
esac
