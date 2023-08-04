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

variable "private_ip" {
  description = "Map with every node's private IP address"
  type = map(string)
  default = {
    controller = "10.1.1.11"
    workbench = "10.1.1.15"
    shell = "10.1.2.17"
    keep0 = "10.1.2.13"
  }
}

variable "dns_aliases" {
  description = "Sets DNS name aliases for every service node"
  type = map(list(string))
  default = {
    workbench = [
      "ws",
      "workbench2",
      "webshell",
      "keep",
      "download",
      "prometheus",
      "grafana",
      "*.collections"
    ]
  }
}

variable "vpc_id" {
  description = "Use existing VPC instead of creating one for the cluster"
  type = string
  default = ""
}

variable "sg_id" {
  description = "Use existing security group instead of creating one for the cluster"
  type = string
  default = ""
}

variable "private_subnet_id" {
  description = "Use existing private subnet instead of creating one for the cluster"
  type = string
  default = ""
}

variable "public_subnet_id" {
  description = "Use existing public subnet instead of creating one for the cluster"
  type = string
  default = ""
}

variable "custom_tags" {
  description = "Apply customized tags to every resource on the cluster"
  type = map(string)
  default = {}
}