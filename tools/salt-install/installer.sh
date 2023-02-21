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
set -o pipefail

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

# The public host used as an SSH jump host
# This will be populated by loadconfig()
declare USE_SSH_JUMPHOST

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
		SSH=`ssh_cmd "$NODE"`
		GIT="eval `git_cmd $NODE`"
		if ! $SSH $DEPLOY_USER@$NODE test -d ${GITTARGET}.git ; then

			# Initialize the git repository (1st time case).  We're
			# actually going to make two repositories here because git
			# will complain if you try to push to a repository with a
			# checkout. So we're going to create a "bare" repository
			# and then clone a regular repository (with a checkout)
			# from that.

			$SSH $DEPLOY_USER@$NODE git init --bare --shared=0600 ${GITTARGET}.git
			if ! $GIT remote add $NODE $DEPLOY_USER@$NODE:${GITTARGET}.git ; then
				$GIT remote set-url $NODE $DEPLOY_USER@$NODE:${GITTARGET}.git
			fi
			$GIT push $NODE $BRANCH
			$SSH $DEPLOY_USER@$NODE "umask 0077 && git clone ${GITTARGET}.git ${GITTARGET}"
		fi

		# The update case.
		#
		# Push to the bare repository on the remote node, then in the
		# remote node repository with the checkout, pull the branch
		# from the bare repository.

		$GIT push $NODE $BRANCH
		$SSH $DEPLOY_USER@$NODE "git -C ${GITTARGET} checkout ${BRANCH} && git -C ${GITTARGET} pull"
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
	SSH=`ssh_cmd "$NODE"`

    if [[ "$NODE" = localhost ]] ; then
	    SUDO=''
		if [[ $(whoami) != 'root' ]] ; then
			SUDO=sudo
		fi
		$SUDO ./provision.sh --config ${CONFIG_FILE} ${ROLES} 2>&1 | tee $logfile
	else
		$SSH $DEPLOY_USER@$NODE "cd ${GITTARGET} && sudo ./provision.sh --config ${CONFIG_FILE} ${ROLES}" 2>&1 | tee $logfile
    fi
}

loadconfig() {
    if [[ ! -s $CONFIG_FILE ]] ; then
		echo "Must be run from initialized setup dir, maybe you need to 'initialize' first?"
    fi
    source ${CONFIG_FILE}
    GITTARGET=arvados-deploy-config-${CLUSTER}
}

ssh_cmd() {
	local NODE=$1
	if [ -z "${USE_SSH_JUMPHOST}" -o "${NODE}" == "${USE_SSH_JUMPHOST}" -o "${NODE}" == "localhost" ]; then
		echo "ssh"
	else
		echo "ssh -J ${DEPLOY_USER}@${USE_SSH_JUMPHOST}"
	fi
}

git_cmd() {
	local NODE=$1
	echo "GIT_SSH_COMMAND=\"`ssh_cmd ${NODE}`\" git"
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
	git init --shared=0600 $SETUPDIR
	cp -r *.sh tests $SETUPDIR

	cp local.params.example.$PARAMS $SETUPDIR/${CONFIG_FILE}
	cp -r config_examples/$SLS $SETUPDIR/${CONFIG_DIR}

	if [[ -n "$TERRAFORM" ]] ; then
	    mkdir $SETUPDIR/terraform
	    cp -r $TERRAFORM/* $SETUPDIR/terraform/
		cp $TERRAFORM/.gitignore $SETUPDIR/terraform/
	fi

	cd $SETUPDIR
	echo '*.log' > .gitignore

	if [[ -n "$TERRAFORM" ]] ; then
		git add terraform
	fi

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
	(cd terraform/vpc && terraform apply -auto-approve) 2>&1 | tee -a $logfile
	(cd terraform/data-storage && terraform apply -auto-approve) 2>&1 | tee -a $logfile
	(cd terraform/services && terraform apply -auto-approve) 2>&1 | grep -v letsencrypt_iam_secret_access_key | tee -a $logfile
	(cd terraform/services && echo -n 'letsencrypt_iam_secret_access_key = ' && terraform output letsencrypt_iam_secret_access_key) 2>&1 | tee -a $logfile
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
	    deploynode $NODE "${NODES[$NODE]}"
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
