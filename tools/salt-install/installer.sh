#!/bin/bash

# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: CC-BY-SA-3.0

#
# installer.sh
#
# Helps manage the configuration in a git repository, and then deploy
# nodes by pushing a copy of the git repository to each node and
# running the provision script to do the actual installation and
# configuration.
#

set -eu

# The parameter file
declare CONFIG_FILE=local.params

# The salt template directory
declare CONFIG_DIR=local_config_dir

# The 5-character Arvados cluster id
# This will be populated by loadconfig()
declare CLUSTER

# The parent domain (not including the cluster id)
# This will be populated by loadconfig()
declare DOMAIN

# A bash associative array listing each node and mapping to the roles
# that should be provisioned on those nodes.
# This will be populated by loadconfig()
declare -A NODES

# The ssh user we'll use
# This will be populated by loadconfig()
declare DEPLOY_USER

# The git repository that we'll push to on all the nodes
# This will be populated by loadconfig()
declare GITTARGET

checktools() {
    local MISSING=''
    for a in git ip ; do
	if ! which $a ; then
	    MISSING="$MISSING $a"
	fi
    done
    if [[ -n "$MISSING" ]] ; then
	echo "Some tools are missing, please make sure you have the 'git' and 'iproute2' packages installed"
	exit 1
    fi
}

sync() {
    local NODE=$1
    local BRANCH=$2

    # Synchronizes the configuration by creating a git repository on
    # each node, pushing our branch, and updating the checkout.

    if [[ "$NODE" != localhost ]] ; then
	if ! ssh $DEPLOY_USER@$NODE test -d ${GITTARGET}.git ; then

	    # Initialize the git repository (1st time case).  We're
	    # actually going to make two repositories here because git
	    # will complain if you try to push to a repository with a
	    # checkout. So we're going to create a "bare" repository
	    # and then clone a regular repository (with a checkout)
	    # from that.

	    ssh $DEPLOY_USER@$NODE git init --bare ${GITTARGET}.git
	    if ! git remote add $NODE $DEPLOY_USER@$NODE:${GITTARGET}.git ; then
			git remote set-url $NODE $DEPLOY_USER@$NODE:${GITTARGET}.git
	    fi
	    git push $NODE $BRANCH
	    ssh $DEPLOY_USER@$NODE git clone ${GITTARGET}.git ${GITTARGET}
	fi

	# The update case.
	#
	# Push to the bare repository on the remote node, then in the
	# remote node repository with the checkout, pull the branch
	# from the bare repository.

	git push $NODE $BRANCH
	ssh $DEPLOY_USER@$NODE "git -C ${GITTARGET} checkout ${BRANCH} && git -C ${GITTARGET} pull"
    fi
}

deploynode() {
    local NODE=$1
    local ROLES=$2

    # Deploy a node.  This runs the provision script on the node, with
    # the appropriate roles.

    if [[ -z "$ROLES" ]] ; then
	echo "No roles specified for $NODE, will deploy all roles"
    else
	ROLES="--roles ${ROLES}"
    fi

    logfile=deploy-${NODE}-$(date -Iseconds).log

    if [[ "$NODE" = localhost ]] ; then
	SUDO=''
	if [[ $(whoami) != 'root' ]] ; then
	    SUDO=sudo
	fi
	$SUDO ./provision.sh --config ${CONFIG_FILE} ${ROLES} 2>&1 | tee $logfile
    else
	ssh $DEPLOY_USER@$NODE "cd ${GITTARGET} && sudo ./provision.sh --config ${CONFIG_FILE} ${ROLES}" 2>&1 | tee $logfile
    fi
}

loadconfig() {
    if [[ ! -s $CONFIG_FILE ]] ; then
	echo "Must be run from initialized setup dir, maybe you need to 'initialize' first?"
    fi
    source ${CONFIG_FILE}
    GITTARGET=arvados-deploy-config-${CLUSTER}
}

set +u
subcmd="$1"
set -u

if [[ -n "$subcmd" ]] ; then
    shift
fi
case "$subcmd" in
    initialize)
	if [[ ! -f provision.sh ]] ; then
	    echo "Must be run from arvados/tools/salt-install"
	    exit
	fi

	checktools

	set +u
	SETUPDIR=$1
	PARAMS=$2
	SLS=$3
	TERRAFORM=$4
	set -u

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

	cp local.params.example.$PARAMS $SETUPDIR/${CONFIG_FILE}
	cp -r config_examples/$SLS $SETUPDIR/${CONFIG_DIR}

	if [[ -n "$TERRAFORM" ]] ; then
	    mkdir $SETUPDIR/terraform
	    cp -r $TERRAFORM/* $SETUPDIR/terraform/
	fi

	cd $SETUPDIR
	echo '*.log' > .gitignore

	git add *.sh ${CONFIG_FILE} ${CONFIG_DIR} tests .gitignore
	git commit -m"initial commit"

	echo
	echo "Setup directory $SETUPDIR initialized."
	if [[ -n "$TERRAFORM" ]] ; then
	    (cd $SETUPDIR/terraform/vpc && terraform init)
	    (cd $SETUPDIR/terraform/data-storage && terraform init)
	    (cd $SETUPDIR/terraform/services && terraform init)
	    echo "Now go to $SETUPDIR, customize 'terraform/vpc/terraform.tfvars' as needed, then run 'installer.sh terraform'"
	else
	    echo "Now go to $SETUPDIR, customize '${CONFIG_FILE}' and '${CONFIG_DIR}' as needed, then run 'installer.sh deploy'"
	fi
	;;

    terraform)
	logfile=terraform-$(date -Iseconds).log
	(cd terraform/vpc && terraform apply) 2>&1 | tee -a $logfile
	(cd terraform/data-storage && terraform apply) 2>&1 | tee -a $logfile
	(cd terraform/services && terraform apply) 2>&1 | tee -a $logfile
	;;

    generate-tokens)
	for i in BLOB_SIGNING_KEY MANAGEMENT_TOKEN SYSTEM_ROOT_TOKEN ANONYMOUS_USER_TOKEN WORKBENCH_SECRET_KEY DATABASE_PASSWORD; do
	    echo ${i}=$(tr -dc A-Za-z0-9 </dev/urandom | head -c 32 ; echo '')
	done
	;;

    deploy)
	set +u
	NODE=$1
	set -u

	checktools

	loadconfig

	if grep -rni 'fixme' ${CONFIG_FILE} ${CONFIG_DIR} ; then
	    echo
	    echo "Some parameters still need to be updated.  Please fix them and then re-run deploy."
	    exit 1
	fi

	BRANCH=$(git branch --show-current)

	set -x

	git add -A
	if ! git diff --cached --exit-code ; then
	    git commit -m"prepare for deploy"
	fi

	if [[ -z "$NODE" ]]; then
	    for NODE in "${!NODES[@]}"
	    do
		# First, push the git repo to each node.  This also
		# confirms that we have git and can log into each
		# node.
		sync $NODE $BRANCH
	    done

	    for NODE in "${!NODES[@]}"
	    do
		# Do 'database' role first,
		if [[ "${NODES[$NODE]}" =~ database ]] ; then
		    deploynode $NODE "${NODES[$NODE]}"
		    unset NODES[$NODE]
		fi
	    done

	    for NODE in "${!NODES[@]}"
	    do
		# then  'api' or 'controller' roles
		if [[ "${NODES[$NODE]}" =~ (api|controller) ]] ; then
		    deploynode $NODE "${NODES[$NODE]}"
		    unset NODES[$NODE]
		fi
	    done

	    for NODE in "${!NODES[@]}"
	    do
		# Everything else (we removed the nodes that we
		# already deployed from the list)
		deploynode $NODE "${NODES[$NODE]}"
	    done
	else
	    # Just deploy the node that was supplied on the command line.
	    sync $NODE $BRANCH
	    deploynode $NODE ""
	fi

	set +x
	echo
	echo "Completed deploy, run 'installer.sh diagnostics' to verify the install"

	;;

    diagnostics)
	loadconfig

	set +u
	declare LOCATION=$1
	set -u

	if ! which arvados-client ; then
	    echo "arvados-client not found, install 'arvados-client' package with 'apt-get' or 'yum'"
	    exit 1
	fi

	if [[ -z "$LOCATION" ]] ; then
	    echo "Need to provide '-internal-client' or '-external-client'"
	    echo
	    echo "-internal-client    You are running this on the same private network as the Arvados cluster (e.g. on one of the Arvados nodes)"
	    echo "-external-client    You are running this outside the private network of the Arvados cluster (e.g. your workstation)"
	    exit 1
	fi

	export ARVADOS_API_HOST="${CLUSTER}.${DOMAIN}:${CONTROLLER_EXT_SSL_PORT}"
	export ARVADOS_API_TOKEN="$SYSTEM_ROOT_TOKEN"

	arvados-client diagnostics $LOCATION
	;;

    *)
	echo "Arvados installer"
	echo ""
	echo "initialize        initialize the setup directory for configuration"
	echo "terraform         create cloud resources using terraform"
	echo "generate-tokens   generate random values for tokens"
	echo "deploy            deploy the configuration from the setup directory"
	echo "diagnostics       check your install using diagnostics"
	;;
esac
