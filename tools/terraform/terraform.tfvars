# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: Apache-2.0

### GENERAL
aws_profile              = "profile-to-use"
aws_region               = "us-east-1"
environment              = "production"
namespace                = "3rd-party deploy test"

### KEYPAIR
key_name                 = "keyname"
key_path                 = "~/.ssh/id_rsa.pub"

cluster                  = "vwxyz"
r53_domain_name          = "vwxyz.arvados.test"

# VPC
# If you have/want to use a VPC already defined, set this value to false
# and uncomment and provide values for the following variables

# manage_vpc               = true
# vpc_id                   = "vpc-12345678901234567"
# private_subnets_ids      = []
# compute_subnets_ids      = []
# public_subnets_ids       = []

cluster_cidr             = "10.0.0.0/16"
azs                      = ["us-east-1a"]
private_subnets          = ["10.0.255.0/24"]
compute_subnets          = ["10.0.254.0/24"]
public_subnets           = ["10.0.0.0/24"]
enable_nat_gateway       = true
enable_vpn_gateway       = false
single_nat_gateway       = true
one_nat_gateway_per_az   = false
enable_dhcp_options      = true

instance_type = {
  "default"   = "m5a.large",
  # "api"       = "m5a.large",
  # "shell"     = "m5a.large",
  # "keepproxy" = "m5a.large",
  # "keepstore" = "m5a.large",
  # "workbench" = "m5a.large",
  # "database"  = "m5a.large",
}
instance_ami = {
  "default"   = "ami-07d02ee1eeb0c996c",
  # "api"       = "ami-07d02ee1eeb0c996c",
  # "shell"     = "ami-07d02ee1eeb0c996c",
  # "keepstore" = "ami-07d02ee1eeb0c996c",
  # "keepproxy" = "ami-07d02ee1eeb0c996c",
  # "workbench" = "ami-07d02ee1eeb0c996c",
  # "database"  = "ami-07d02ee1eeb0c996c",
}

data_bd_size = {
  "default"   = 50,
  # "api"       = 50,
  # "shell"     = 50,
  # "keepproxy" = 50,
  # "keepstore" = 50,
  # "workbench" = 50,
   "database"  = 250,
}
# KEEPSTORE/s
keepstore_count = 2

# SECURITY
# CIDRs allowed unrestricted access to the instances
# allowed_access_cidrs = "0.0.0.0/0"

# If you have/want to use already defined security groups, set this value to false
# and uncomment and provide values for the following variables
# vpc_security_group_ids = {
#   "default"    = "sg-01111111111111110",
#   "ssh"        = "sg-01234567890123456",
#   "http"       = "sg-12345678901234567",
#   "https"      = "sg-23456789012345678",
#   "webshell"   = "sg-34567890123456789",
#   "postgresql" = "sg-45678901234567890",
#   "keepstore"  = "sg-56789012345678901",
# }
