# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: CC-BY-SA-3.0

# Main cluster configurations. No sensible defaults provided for these:
# region_name = "us-east-1"
# cluster_name = "xarv1"
# domain_name = "xarv1.example.com"

# Uncomment this to create an non-publicly accessible Arvados cluster
# private_only = true

# Optional networking options. Set existing resources to be used instead of
# creating new ones.
# NOTE: We only support fully managed or fully custom networking, not a mix of both.
#
# vpc_id = "vpc-aaaa"
# sg_id = "sg-bbbb"
# public_subnet_id = "subnet-cccc"
# private_subnet_id = "subnet-dddd"
#
# RDS related parameters:
# use_rds = true
# additional_rds_subnet_id = "subnet-eeee"

# Optional custom tags to add to every resource. Default: {}
# custom_tags = {
#   environment = "production"
#   project = "Phoenix"
#   owner = "jdoe"
# }

# Optional cluster service nodes configuration:
#
# List of node names which either will be hosting user-facing or internal
# services. Defaults:
# user_facing_hosts = [ "controller", "workbench" ]
# internal_service_hosts = [ "keep0", "shell" ]
#
# Map assigning each node name an internal IP address. Defaults:
# private_ip = {
#   controller = "10.1.1.11"
#   workbench = "10.1.1.15"
#   shell = "10.1.2.17"
#   keep0 = "10.1.2.13"
# }
#
# Map assigning DNS aliases for service node names. Defaults:
# dns_aliases = {
#   workbench = [
#     "ws",
#     "workbench2",
#     "webshell",
#     "keep",
#     "download",
#     "prometheus",
#     "grafana",
#     "*.collections"
#   ]
# }