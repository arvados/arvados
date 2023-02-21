# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: CC-BY-SA-3.0

variable "default_instance_type" {
  description = "The default EC2 instance type to use on the nodes"
  type = string
  default = "m5a.large"
}

variable "pubkey_path" {
  description = "Path to the file containing the public SSH key"
  type = string
  default = "~/.ssh/id_rsa.pub"
}

variable "ssl_password_secret_name_suffix" {
  description = "Name suffix for the SSL certificate's private key password AWS secret."
  type = string
  default = "arvados-ssl-privkey-password"
}