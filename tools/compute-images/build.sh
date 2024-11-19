#!/bin/bash

# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: Apache-2.0

read -rd "\000" helpmessage <<EOF
$(basename $0): Build cloud images for arvados-dispatch-cloud

Syntax:
        $(basename $0) [options]

Options:

  --json-file <path>
      Path to the packer json file (required)
  --arvados-cluster-id <xxxxx>
      The ID of the Arvados cluster, e.g. zzzzz(required)
  --aws-profile <profile>
      AWS profile to use (valid profile from ~/.aws/config (optional)
  --aws-secrets-file <path>
      AWS secrets file which will be sourced from this script (optional)
      When building for AWS, either an AWS profile or an AWS secrets file
      must be provided.
  --aws-source-ami <ami-xxxxxxxxxxxxxxxxx>
      The AMI to use as base for building the images (required if building for AWS)
  --aws-region <region> (default: us-east-1)
      The AWS region to use for building the images
  --aws-vpc-id <vpc-id>
      VPC id for AWS, if not specified packer will derive from the subnet id or pick the default one.
  --aws-subnet-id <subnet-xxxxxxxxxxxxxxxxx>
      Subnet id for AWS, if not specified packer will pick the default one for the VPC.
  --aws-ebs-autoscale, --no-aws-ebs-autoscale
      These flags determine whether or not to use an EBS autoscaling volume for
      Crunch's working directory. The default is to use this when building an
      image on AWS.
  --aws-associate-public-ip <true|false>
      Associate a public IP address with the node used for building the compute image.
      Required when the machine running packer can not reach the node used for building
      the compute image via its private IP. (default: true if building for AWS)
      Note: if the subnet has "Auto-assign public IPv4 address" enabled, disabling this
      flag will have no effect.
  --aws-ena-support <true|false>
      Enable enhanced networking (default: true if building for AWS)
  --gcp-project-id <project-id>
      GCP project id (required if building for GCP)
  --gcp-account-file <path>
      GCP account file (required if building for GCP)
  --gcp-zone <zone> (default: us-central1-f)
      GCP zone
  --azure-secrets-file <patch>
      Azure secrets file which will be sourced from this script (required if building for Azure)
  --azure-resource-group <resouce-group>
      Azure resource group (required if building for Azure)
  --azure-location <location>
      Azure location, e.g. centralus, eastus, westeurope (required if building for Azure)
  --azure-sku <sku> (required if building for Azure, e.g. 16.04-LTS)
      Azure SKU image to use
  --ssh_user <user> (default: packer)
      The user packer will use to log into the image
  --workdir <path> (default: /tmp)
      The directory where data files are staged and setup scripts are run
  --resolver <resolver_IP>
      The dns resolver for the machine (default: host's network provided)
  --reposuffix <suffix>
      Set this to "-dev" to track the unstable/dev Arvados repositories
  --pin-packages, --no-pin-packages
      These flags determine whether or not to configure apt pins for Arvados
      and third-party packages it depends on. By default packages are pinned
      unless you set \`--reposuffix -dev\`.
  --public-key-file <path>
      Path to the public key file that a-d-c will use to log into the compute node (required)
  --mksquashfs-mem (default: 256M)
      Only relevant when using Singularity. This is the amount of memory mksquashfs is allowed to use.
  --nvidia-gpu-support
      Install all the necessary tooling for Nvidia GPU support (default: do not install Nvidia GPU support)
  --debug
      Output debug information (default: no debug output is printed)

For more information, see the Arvados documentation at https://doc.arvados.org/install/crunch2-cloud/install-compute-node.html

EOF

set -e -o pipefail

ansible_vars_file="$(mktemp --tmpdir ansible-vars-XXXXXX.yml)"
trap 'rm -f "$ansible_vars_file"' EXIT INT TERM QUIT
# FIXME? We build the compute node image with the same version of Go that
# Arvados uses, but it's not clear that we should: the only thing we use Go
# for is to build Singularity, so what matters is what Singularity wants, not
# what Arvados wants.
sed -rn 's/^const +goversion *= */compute_go_version: /p' \
    <../../lib/install/deps.go >>"$ansible_vars_file"

ansible_set_var() {
    eval "$(printf "%s=%q" "$1" "$2")"
    echo "$1: $2" >>"$ansible_vars_file"
}

JSON_FILE=
ARVADOS_CLUSTER_ID=
AWS_PROFILE=
AWS_SECRETS_FILE=
AWS_SOURCE_AMI=
AWS_VPC_ID=
AWS_SUBNET_ID=
AWS_EBS_AUTOSCALE=
AWS_ASSOCIATE_PUBLIC_IP=true
AWS_ENA_SUPPORT=true
GCP_PROJECT_ID=
GCP_ACCOUNT_FILE=
GCP_ZONE=
AZURE_SECRETS_FILE=
AZURE_RESOURCE_GROUP=
AZURE_LOCATION=
AZURE_CLOUD_ENVIRONMENT=
DEBUG=
SSH_USER=
AWS_DEFAULT_REGION=us-east-1

PARSEDOPTS=$(getopt --name "$0" --longoptions \
    help,json-file:,arvados-cluster-id:,aws-source-ami:,aws-profile:,aws-secrets-file:,aws-region:,aws-vpc-id:,aws-subnet-id:,aws-ebs-autoscale,no-aws-ebs-autoscale,aws-associate-public-ip:,aws-ena-support:,gcp-project-id:,gcp-account-file:,gcp-zone:,azure-secrets-file:,azure-resource-group:,azure-location:,azure-sku:,azure-cloud-environment:,ssh_user:,workdir:,resolver:,reposuffix:,pin-packages,no-pin-packages,public-key-file:,mksquashfs-mem:,nvidia-gpu-support,debug \
    -- "" "$@")
if [ $? -ne 0 ]; then
    exit 1
fi

eval set -- "$PARSEDOPTS"
while [ $# -gt 0 ]; do
    case "$1" in
        --help)
            echo >&2 "$helpmessage"
            echo >&2
            exit 1
            ;;
        --json-file)
            JSON_FILE="$2"; shift
            ;;
        --arvados-cluster-id)
            ARVADOS_CLUSTER_ID="$2"; shift
            ;;
        --aws-source-ami)
            AWS_SOURCE_AMI="$2"; shift
            ;;
        --aws-profile)
            AWS_PROFILE="$2"; shift
            ;;
        --aws-secrets-file)
            AWS_SECRETS_FILE="$2"; shift
            ;;
        --aws-region)
            AWS_DEFAULT_REGION="$2"; shift
            ;;
        --aws-vpc-id)
            AWS_VPC_ID="$2"; shift
            ;;
        --aws-subnet-id)
            AWS_SUBNET_ID="$2"; shift
            ;;
        --aws-ebs-autoscale)
            ansible_set_var arvados_compute_encrypted_tmp aws_ebs
            ;;
        --no-aws-ebs-autoscale)
            ansible_set_var arvados_compute_encrypted_tmp '""'
            ;;
        --aws-associate-public-ip)
            AWS_ASSOCIATE_PUBLIC_IP="$2"; shift
            ;;
        --aws-ena-support)
            AWS_ENA_SUPPORT="$2"; shift
            ;;
        --gcp-project-id)
            GCP_PROJECT_ID="$2"; shift
            ;;
        --gcp-account-file)
            GCP_ACCOUNT_FILE="$2"; shift
            ;;
        --gcp-zone)
            GCP_ZONE="$2"; shift
            ;;
        --azure-secrets-file)
            AZURE_SECRETS_FILE="$2"; shift
            ;;
        --azure-resource-group)
            AZURE_RESOURCE_GROUP="$2"; shift
            ;;
        --azure-location)
            AZURE_LOCATION="$2"; shift
            ;;
        --azure-sku)
            AZURE_SKU="$2"; shift
            ;;
        --azure-cloud-environment)
            AZURE_CLOUD_ENVIRONMENT="$2"; shift
            ;;
        --ssh_user)
            SSH_USER="$2"; shift
            ;;
        --workdir)
            ansible_set_var workdir "$2"; shift
            ;;
        --resolver)
            ansible_set_var dns_resolver "$2"; shift
            ;;
        --reposuffix)
            ansible_set_var arvados_apt_suites "$2"; shift
            ;;
        --pin-packages)
            ansible_set_var arvados_compute_pin_packages true
            ;;
        --no-pin-packages)
            ansible_set_var arvados_pin_version '""'
            ansible_set_var arvados_compute_pin_packages false
            ;;
        --public-key-file)
            ansible_set_var compute_authorized_keys "$(readlink -e "$2")"; shift
            ;;
        --mksquashfs-mem)
            ansible_set_var compute_mksquashfs_mem "$2"; shift
            ;;
        --nvidia-gpu-support)
            ansible_set_var arvados_compute_nvidia true
            ;;
        --debug)
            # If you want to debug a build issue, add the -debug flag to the build
            # command in question.
            # This will allow you to ssh in, if you use the .pem file that packer
            # generates in this directory as the ssh key. The base image uses the admin
            # user and ssh port 22.
            EXTRA=" -debug"
            ;;
        --)
            if [ $# -gt 1 ]; then
                echo >&2 "$0: unrecognized argument '$2'. Try: $0 --help"
                exit 1
            fi
            ;;
    esac
    shift
done

if [[ -z "$JSON_FILE" ]] || [[ ! -f "$JSON_FILE" ]]; then
  echo >&2 "$helpmessage"
  echo >&2
  echo >&2 "ERROR: packer json file not found"
  echo >&2
  exit 1
fi

if [[ -z "$ARVADOS_CLUSTER_ID" ]]; then
  echo >&2 "$helpmessage"
  echo >&2
  echo >&2 "ERROR: arvados cluster id not specified"
  echo >&2
  exit 1
fi

if [[ -z "${compute_authorized_keys:-}" || ! -f "$compute_authorized_keys" ]]; then
  echo >&2 "$helpmessage"
  echo >&2
  echo >&2 "ERROR: public key file file not found"
  echo >&2
  exit 1
fi

if [[ ! -z "$AWS_SECRETS_FILE" ]]; then
  source $AWS_SECRETS_FILE
fi

if [[ ! -z "$AZURE_SECRETS_FILE" ]]; then
  source $AZURE_SECRETS_FILE
fi

AWS=0
EXTRA2=""

if [[ -n "$AWS_SOURCE_AMI" ]]; then
  EXTRA2+=" -var aws_source_ami=$AWS_SOURCE_AMI"
  AWS=1
fi
if [[ -n "$AWS_PROFILE" ]]; then
  EXTRA2+=" -var aws_profile=$AWS_PROFILE"
  AWS=1
fi
if [[ -n "$AWS_VPC_ID" ]]; then
  EXTRA2+=" -var vpc_id=$AWS_VPC_ID"
  AWS=1
fi
if [[ -n "$AWS_SUBNET_ID" ]]; then
  EXTRA2+=" -var subnet_id=$AWS_SUBNET_ID"
  AWS=1
fi
if [[ -n "$AWS_DEFAULT_REGION" ]]; then
  EXTRA2+=" -var aws_default_region=$AWS_DEFAULT_REGION"
  AWS=1
fi
if [[ $AWS -eq 1 ]]; then
  EXTRA2+=" -var aws_associate_public_ip_address=$AWS_ASSOCIATE_PUBLIC_IP"
  EXTRA2+=" -var aws_ena_support=$AWS_ENA_SUPPORT"
fi
if [[ -n "$GCP_PROJECT_ID" ]]; then
  EXTRA2+=" -var project_id=$GCP_PROJECT_ID"
fi
if [[ -n "$GCP_ACCOUNT_FILE" ]]; then
  EXTRA2+=" -var account_file=$GCP_ACCOUNT_FILE"
fi
if [[ -n "$GCP_ZONE" ]]; then
  EXTRA2+=" -var zone=$GCP_ZONE"
fi
if [[ -n "$AZURE_RESOURCE_GROUP" ]]; then
  EXTRA2+=" -var resource_group=$AZURE_RESOURCE_GROUP"
fi
if [[ -n "$AZURE_LOCATION" ]]; then
  EXTRA2+=" -var location=$AZURE_LOCATION"
fi
if [[ -n "$AZURE_SKU" ]]; then
  EXTRA2+=" -var image_sku=$AZURE_SKU"
fi
if [[ -n "$AZURE_CLOUD_ENVIRONMENT" ]]; then
  EXTRA2+=" -var cloud_environment_name=$AZURE_CLOUD_ENVIRONMENT"
fi
if [[ -n "$SSH_USER" ]]; then
  EXTRA2+=" -var ssh_user=$SSH_USER"
fi
if [[ -z "${arvados_compute_pin_packages:-}" && "${arvados_apt_suites:-}" = -dev ]]; then
    ansible_set_var arvados_pin_version '""'
    ansible_set_var arvados_compute_pin_packages false
fi

logfile=packer-$(date -Iseconds).log

echo
cat "$ansible_vars_file"
packer version
echo
echo packer build$EXTRA -var "arvados_cluster=$ARVADOS_CLUSTER_ID" -var "ansible_vars_file=$ansible_vars_file" $EXTRA2 $JSON_FILE | tee -a $logfile
packer build$EXTRA -var "arvados_cluster=$ARVADOS_CLUSTER_ID" -var "ansible_vars_file=$ansible_vars_file" $EXTRA2 $JSON_FILE 2>&1 | tee -a $logfile
