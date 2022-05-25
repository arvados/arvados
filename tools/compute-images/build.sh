#!/bin/bash

# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: Apache-2.0

JSON_FILE=$1
ARVADOS_CLUSTER=$2
PROJECT_ID=$3
ACCOUNT_FILE=$4

read -rd "\000" helpmessage <<EOF
$(basename $0): Build cloud images for arvados-dispatch-cloud

Syntax:
        $(basename $0) [options]

Options:

  --json-file (required)
      Path to the packer json file
  --arvados-cluster-id (required)
      The ID of the Arvados cluster, e.g. zzzzz
  --aws-profile (default: false)
      AWS profile to use (valid profile from ~/.aws/config
  --aws-secrets-file (default: false, required if building for AWS)
      AWS secrets file which will be sourced from this script
  --aws-source-ami (default: false, required if building for AWS)
      The AMI to use as base for building the images
  --aws-region (default: us-east-1)
      The AWS region to use for building the images
  --aws-vpc-id (optional)
      VPC id for AWS, otherwise packer will pick the default one
  --aws-subnet-id
      Subnet id for AWS otherwise packer will pick the default one for the VPC
  --aws-ebs-autoscale (default: false)
      Install the AWS EBS autoscaler daemon.
  --aws-associate-public-ip (default: true if building for AWS)
      Associate a public IP address with the node used for building the compute image.
      Required when the machine running packer can not reach the node used for building
      the compute image via its private IP.
      Note: if the subnet has "Auto-assign public IPv4 address" enabled, disabling this
      flag will have no effect.
  --aws-ena-support (default: true if building for AWS)
      Enable enhanced networking
  --gcp-project-id (default: false, required if building for GCP)
      GCP project id
  --gcp-account-file (default: false, required if building for GCP)
      GCP account file
  --gcp-zone (default: us-central1-f)
      GCP zone
  --azure-secrets-file (default: false, required if building for Azure)
      Azure secrets file which will be sourced from this script
  --azure-resource-group (default: false, required if building for Azure)
      Azure resource group
  --azure-location (default: false, required if building for Azure)
      Azure location, e.g. centralus, eastus, westeurope
  --azure-sku (default: unset, required if building for Azure, e.g. 16.04-LTS)
      Azure SKU image to use
  --ssh_user  (default: packer)
      The user packer will use to log into the image
  --resolver (default: host's network provided)
      The dns resolver for the machine
  --reposuffix (default: unset)
      Set this to "-dev" to track the unstable/dev Arvados repositories
  --public-key-file (required)
      Path to the public key file that a-d-c will use to log into the compute node
  --mksquashfs-mem (default: 256M)
      Only relevant when using Singularity. This is the amount of memory mksquashfs is allowed to use.
  --nvidia-gpu-support (default: false)
      Install all the necessary tooling for Nvidia GPU support
  --debug (default: false)
      Output debug information

For more information, see the Arvados documentation at https://doc.arvados.org/install/crunch2-cloud/install-compute-node.html

EOF

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
PUBLIC_KEY_FILE=
MKSQUASHFS_MEM=256M
NVIDIA_GPU_SUPPORT=

PARSEDOPTS=$(getopt --name "$0" --longoptions \
    help,json-file:,arvados-cluster-id:,aws-source-ami:,aws-profile:,aws-secrets-file:,aws-region:,aws-vpc-id:,aws-subnet-id:,aws-ebs-autoscale,aws-associate-public-ip:,aws-ena-support:,gcp-project-id:,gcp-account-file:,gcp-zone:,azure-secrets-file:,azure-resource-group:,azure-location:,azure-sku:,azure-cloud-environment:,ssh_user:,resolver:,reposuffix:,public-key-file:,mksquashfs-mem:,nvidia-gpu-support,debug \
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
            AWS_EBS_AUTOSCALE=1
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
        --resolver)
            RESOLVER="$2"; shift
            ;;
        --reposuffix)
            REPOSUFFIX="$2"; shift
            ;;
        --public-key-file)
            PUBLIC_KEY_FILE="$2"; shift
            ;;
        --mksquashfs-mem)
            MKSQUASHFS_MEM="$2"; shift
            ;;
        --nvidia-gpu-support)
            NVIDIA_GPU_SUPPORT=1
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

if [[ -z "$PUBLIC_KEY_FILE" ]] || [[ ! -f "$PUBLIC_KEY_FILE" ]]; then
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
if [[ -n "$AWS_EBS_AUTOSCALE" ]]; then
  EXTRA2+=" -var aws_ebs_autoscale=$AWS_EBS_AUTOSCALE"
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
if [[ -n "$RESOLVER" ]]; then
  EXTRA2+=" -var resolver=$RESOLVER"
fi
if [[ -n "$REPOSUFFIX" ]]; then
  EXTRA2+=" -var reposuffix=$REPOSUFFIX"
fi
if [[ -n "$PUBLIC_KEY_FILE" ]]; then
  EXTRA2+=" -var public_key_file=$PUBLIC_KEY_FILE"
fi
if [[ -n "$MKSQUASHFS_MEM" ]]; then
  EXTRA2+=" -var mksquashfs_mem=$MKSQUASHFS_MEM"
fi
if [[ -n "$NVIDIA_GPU_SUPPORT" ]]; then
  EXTRA2+=" -var nvidia_gpu_support=$NVIDIA_GPU_SUPPORT"
fi

GOVERSION=$(grep 'const goversion =' ../../lib/install/deps.go |awk -F'"' '{print $2}')
EXTRA2+=" -var goversion=$GOVERSION"

echo
packer version
echo
echo packer build$EXTRA -var "arvados_cluster=$ARVADOS_CLUSTER_ID"$EXTRA2 $JSON_FILE
packer build$EXTRA -var "arvados_cluster=$ARVADOS_CLUSTER_ID"$EXTRA2 $JSON_FILE
