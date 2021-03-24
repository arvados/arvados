# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: Apache-2.0

module "vpc" {
  count                    = var.manage_vpc ? 1 : 0
  source                   = "terraform-aws-modules/vpc/aws"
  version                  = "2.77.0"

  name                     = var.cluster
  cidr                     = var.cluster_cidr

  azs                      = var.azs
  # we'll need internet access in the compute nodes, so the compute_subnets will
  # go to the private_subnets
  private_subnets          = concat(var.private_subnets, var.compute_subnets)
  public_subnets           = var.public_subnets

  enable_dns_hostnames     = true
  enable_dns_support       = true

  enable_s3_endpoint       = true

  enable_nat_gateway       = var.enable_nat_gateway
  enable_vpn_gateway       = var.enable_vpn_gateway
  single_nat_gateway       = var.single_nat_gateway
  one_nat_gateway_per_az   = var.one_nat_gateway_per_az

  enable_dhcp_options      = var.enable_dhcp_options
  dhcp_options_domain_name = var.r53_domain_name
  tags                     = local.resource_tags
}
