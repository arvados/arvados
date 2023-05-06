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

variable "private_only" {
  description = "Don't create infrastructure reachable from the public Internet"
  type = bool
  default = false
}

variable "user_facing_hosts" {
  description = "List of hostnames for nodes that hold user-accesible Arvados services"
  type = list(string)
  default = [ "controller", "workbench" ]
}

variable "internal_service_hosts" {
  description = "List of hostnames for nodes that hold internal Arvados services"
  type = list(string)
  default = [ "keep0", "shell" ]
}