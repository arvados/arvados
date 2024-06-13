# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: CC-BY-SA-3.0

variable "instance_type" {
  description = "The EC2 instance types to use per service node"
  type = map(string)
  default = {
    default = "m5a.large"
  }
}

variable "instance_volume_size" {
  description = "EC2 volume size in GiB per service node"
  type = map(number)
  default = {
    default = 20
    controller = 100
  }
}

variable "pubkey_path" {
  description = "Path to the file containing the public SSH key"
  type = string
  default = "~/.ssh/id_rsa.pub"
}

variable "deploy_user" {
  description = "User for deploying the software"
  type = string
  default = "admin"
}

variable "ssl_password_secret_name_suffix" {
  description = "Name suffix for the SSL certificate's private key password AWS secret."
  type = string
  default = "arvados-ssl-privkey-password"
}

variable "instance_ami" {
  description = "The EC2 instance AMI to use on the nodes"
  type = string
  default = ""
}

variable "use_rds" {
  description = "Enable to create an RDS instance as the cluster's database service"
  type = bool
  default = false
}

variable "rds_username" {
  description = "RDS instance's username. Default: <cluster_name>_arvados"
  type = string
  default = ""
}

variable "rds_password" {
  description = "RDS instance's password. Default: randomly-generated 32 chars"
  type = string
  default = ""
}

variable "rds_instance_type" {
  description = "RDS instance type"
  type = string
  default = "db.m5.large"
}

variable "rds_allocated_storage" {
  description = "RDS initial storage size (GiB)"
  type = number
  default = 60
}

variable "rds_max_allocated_storage" {
  description = "RDS maximum storage size that will autoscale to (GiB)"
  type = number
  default = 300
}
