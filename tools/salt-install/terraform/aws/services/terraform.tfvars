# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: CC-BY-SA-3.0

# SSH public key path to use by the installer script. It will be installed in
# the home directory of the 'deploy_user'. Default: ~/.ssh/id_rsa.pub
# pubkey_path = "/path/to/pub.key"

# Set the instance type for your nodes. Default: m5a.large
# instance_type = {
#   default = "m5a.xlarge"
#   controller = "c5a.4xlarge"
# }

# Set the volume size (in GiB) per service node.
# Default: 100 for controller, 20 the rest.
# NOTE: The service node will need to be rebooted after increasing its volume's
# size.
# instance_volume_size = {
#   default = 20
#   controller = 300
# }

# Use an RDS instance for database. For this to work, make sure to also set
# 'use_rds' to true in '../vpc/terraform.tfvars'.
# use_rds = true
#
# Provide custom values if needed.
# rds_username = ""
# rds_password = ""
# rds_instance_type = "db.m5.xlarge"
# rds_postgresql_version = "16.3"
# rds_allocated_storage = 200
# rds_max_allocated_storage = 1000
# rds_backup_retention_period = 30
# rds_backup_before_deletion = false
# rds_final_backup_name = ""

# AWS secret's name which holds the SSL certificate private key's password.
# Default: "arvados-ssl-privkey-password"
# ssl_password_secret_name_suffix = "some-name-suffix"

# User for software deployment. Depends on the AMI's distro.
# Default: "admin"
# deploy_user = "ubuntu"

# Instance AMI to use for service nodes. Default: latest from Debian 11
# instance_ami = "ami-0481e8ba7f486bd99"