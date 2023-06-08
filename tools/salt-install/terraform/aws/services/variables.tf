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