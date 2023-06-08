# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: CC-BY-SA-3.0

output "arvados_vpc_id" {
  value = local.arvados_vpc_id
}
output "arvados_vpc_cidr" {
  value = try(local.arvados_vpc_cidr_block, "")
}

output "public_subnet_id" {
  value = local.public_subnet_id
}

output "private_subnet_id" {
  value = local.private_subnet_id
}

output "arvados_sg_id" {
  value = local.arvados_sg_id
}

output "eip_id" {
  value = { for k, v in aws_eip.arvados_eip: k => v.id }
}

output "public_ip" {
  value = local.public_ip
}

output "public_hosts" {
  value = local.public_hosts
}

output "private_ip" {
  value = local.private_ip
}

output "private_hosts" {
  value = local.private_hosts
}

output "user_facing_hosts" {
  value = var.user_facing_hosts
}

output "internal_service_hosts" {
  value = var.internal_service_hosts
}

output "private_only" {
  value = var.private_only
}

output "route53_dns_ns" {
  value = try(local.route53_public_zone.name_servers, [])
}

output "letsencrypt_iam_access_key_id" {
  value = try(local.iam_access_key_letsencrypt.id, "")
  sensitive = true
}

output "letsencrypt_iam_secret_access_key" {
  value = try(local.iam_access_key_letsencrypt.secret, "")
  sensitive = true
}

output "region_name" {
  value = var.region_name
}

output "cluster_name" {
  value = var.cluster_name
}

output "domain_name" {
  value = var.domain_name
}

output "custom_tags" {
  value = var.custom_tags
}
