# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: Apache-2.0

output "vpc_id" {
  value = var.manage_vpc ? module.vpc.0.vpc_id : var.vpc_id
}
output "cluster" {
  value = var.cluster
}
output "vpc_name" {
  value = var.manage_vpc ? module.vpc.0.name : var.cluster
}
output "vpc_cidr" {
  value = var.cluster_cidr
}
output "vpc_azs" {
  value = var.manage_vpc ? module.vpc.0.azs : var.azs
}
output "vpc_private_subnets_ids" {
  value = var.manage_vpc ? module.vpc.0.private_subnets : concat(var.private_subnets_ids, var.compute_subnets_ids)
}
output "vpc_compute_subnets_ids" {
  value = var.manage_vpc ? [module.vpc.0.private_subnets[1]] : var.compute_subnets_ids
}
output "vpc_public_subnets_ids" {
  value = var.manage_vpc ? module.vpc.0.public_subnets : var.public_subnets_ids
}
output "vpc_nat_public_ips" {
  value = var.manage_vpc ? module.vpc.0.nat_public_ips : null
}
output "vpc_private_subnets_cidr_blocks" {
  value = var.manage_vpc ? module.vpc.0.private_subnets_cidr_blocks : concat(var.private_subnets, var.compute_subnets)
}
output "vpc_compute_subnets_cidr_blocks" {
  value = var.manage_vpc ? [module.vpc.0.private_subnets_cidr_blocks[1]] : var.compute_subnets
}
output "vpc_public_subnets_cidr_blocks" {
  value = var.manage_vpc ? module.vpc.0.public_subnets_cidr_blocks : var.public_subnets
}
