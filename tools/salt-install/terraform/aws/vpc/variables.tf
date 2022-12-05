# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: CC-BY-SA-3.0

variable "region_name" {
  description = "Name of the AWS Region where to install Arvados"
  type = string
}

variable "cluster_name" {
  description = "A 5-char alphanum identifier for your Arvados cluster"
  type = string
  validation {
    condition = length(var.cluster_name) == 5
    error_message = "cluster_name should be 5 chars long."
  }
}

variable "domain_name" {
  description = "The domain name under which your Arvados cluster will be hosted"
  type = string
}