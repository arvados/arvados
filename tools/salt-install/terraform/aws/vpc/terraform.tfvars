# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: CC-BY-SA-3.0

region_name = "us-east-1"
# cluster_name = "xarv1"
# domain_name = "xarv1.example.com"

# Uncomment this to create an non-publicly accessible Arvados cluster
# private_only = true

# Optional networking options. Set existing resources to be used instead of
# creating new ones.
# NOTE: We only support fully managed or fully custom networking, not a mix of both.
# vpc_id = "vpc-"
# sg_id = "sg-"
# public_subnet_id = "subnet-"
# private_subnet_id = "subnet-"

# Optional custom tags to add to every resource. Default: {}
# custom_tags = {
#   environment = "production"
#   project = "Phoenix"
#   owner = "jdoe"
# }

# Optional cluster service nodes configuration:
#
# List of node names which either will be hosting user-facing or internal services
# user_facing_hosts = [...]
# internal_service_hosts = [...]
#
# Map assigning each node name an internal IP address
# private_ip = {...}