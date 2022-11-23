# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: CC-BY-SA-3.0

output "arvados_vpc_id" {
  value = aws_vpc.arvados_vpc.id
}
output "arvados_vpc_cidr" {
  value = aws_vpc.arvados_vpc.cidr_block
}

output "arvados_subnet_id" {
  value = aws_subnet.arvados_subnet.id
}

output "arvados_sg_id" {
  value = aws_security_group.arvados_sg.id
}

output "eip_id" {
  value = { for k, v in aws_eip.arvados_eip: k => v.id }
}

output "public_ip" {
  value = local.public_ip
}

output "private_ip" {
  value = local.private_ip
}

output "route53_dns_ns" {
  value = aws_route53_zone.public_zone.name_servers
}

output "letsencrypt_iam_access_key_id" {
  value = aws_iam_access_key.letsencrypt.id
}

output "letsencrypt_iam_secret_access_key" {
  value = aws_iam_access_key.letsencrypt.secret
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
