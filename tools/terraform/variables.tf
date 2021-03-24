# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: Apache-2.0

### GENERAL
variable "aws_region" {
  description = "The AWS region where to deploy the cluster"
  type        = string
  default     = ""
}
variable "aws_profile" {
  description = "The AWS profile to use"
  type        = string
  default     = ""
}
variable "ami" {
  description = "The AMI to use when launching instances"
  # This is Debian buster in us-east-1
  type        = string
  default     = "ami-07d02ee1eeb0c996c"
}
variable "namespace" {
  description = "A descriptive name for the resources' namespace"
  type        = string
  default     = ""
}
variable "environment" {
  description = "A descriptive name for the cluster's environment"
  type        = string
  default     = ""
}
variable "tags" {
  description = "Tags to add to all the environment resources, beside the default ones (Environment, Namespace, Terraform)"
  type        = map
  default     = {}
}

### KEYPAIRS
variable "key_path" {
  description = "Where the keypair is localted (e.g. `~/.ssh/id_rsa.pub`)"
  type        = string
  default     = "" 
}
variable "key_name" {
  description = "Name to give to the keypair"
  type        = string
  default     = ""
}
variable "key_public_key" {
  description = "Public key value if you didn't provide a path to the key"
  type        = string
  default     = ""
}
variable "enable_key_pair" {
  description = "A boolean flag to enable/disable key pair"
  type        = bool
  default     = true
}

### VPC
variable "cluster" {
  type    = string
}
variable "cluster_cidr" {
  type = string
  default = ""
}
variable "azs" {
  type = list(string)
  default = []
}
variable "private_subnets" {
  type = list(string)
  default = []
}
variable "public_subnets" {
  type = list(string)
  default = []
}
variable "compute_subnets" {
  type = list(string)
  default = []
}
variable "private_subnets_ids" {
  type = list(string)
  default = []
}
variable "public_subnets_ids" {
  type = list(string)
  default = []
}
variable "compute_subnets_ids" {
  type = list(string)
  default = []
}
variable "enable_nat_gateway" {
  type = bool
  default = true
}
variable "enable_vpn_gateway" {
  type = bool
  default = false
}
variable "single_nat_gateway" {
  type = bool
  default = true
}
variable "one_nat_gateway_per_az" {
  type = bool
  default = false
}
variable "route53_force_destroy" {
  description = "Destroy R53 zone when VPC is destroyed"
  type = bool
  default = false
}
variable "enable_dhcp_options" {
  type = bool
  default = false
}
variable "allowed_http_access_cidrs" {
  description = "CIDRs that will have HTTP/HTTPs access to the cluster's VPC instances"
  type        = list(string)
  default     = ["0.0.0.0/0"]
}
variable "allowed_ssh_access_cidrs" {
  description = "CIDRs that will have ssh access to the cluster's VPC instances"
  type        = list(string)
  default     = []
}
variable "allowed_full_access_cidrs" {
  description = "CIDRs that will have full access to the cluster's VPC instances"
  type        = list(string)
  default     = []
}

variable "kms_key_id" {
  default = ""
}

variable "key_pair" {
  default = ""
}

variable "instance_ami" {
  type    = map(string)
  default = {}
}

variable "instance_type" {
  type    = map(string)
  default = {}
}

variable "arvados_volume_name" {
  type    = string
  default = ""
}

variable "keepstore_count" {
  type    = string
  default = "1"
}

variable "root_bd_size" {
  type    = number
  default = 50
}

variable "data_bd_size" {
  type    = map(number)
  default = {}
}

variable "db_identifier" {
  type    = string
  default = ""
}

variable "db_engine_version" {
  type    = string
  default = "11.8"
}

variable "db_instance_class" {
  type    = string
  default = "db.t2.large"
}

variable "db_allocated_storage" {
  type    = number
  default = 512
}

variable "db_storage_encrypted" {
  type    = bool
  default = true
}

variable "db_permissions_boundary" {
  type    = string
  default = ""
}

variable "manage_vpc" {
  description = "If you want to manage/create the VPC where the cluster will be deployed with terraform"
  type    = bool
  default = true
}

variable "vpc_id" {
  description = "If you are not managing the vpc with terraform, then you need to provide the VPC ID"
  type    = string
  default = ""
}

variable "manage_security_groups" {
  description = "If you want to manage/create the security groups with terraform"
  type    = bool
  default = true
}

variable "vpc_security_group_ids" {
  description = "If you are not managing the security groups, then you need to provide them"
  type    = map(string)
  default = {}
}

variable "vpc_cidr" {
  type    = string
  default = ""

}
variable "vpc_subnet_cidrs" {
  type    = list(string)
  default = []
}
variable "r53_domain_name" {
  description = "Domain which will be appended to the cluster name"
  type    = string
  default = ""
}

