# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: CC-BY-SA-3.0

output "vpc_id" {
  value = data.terraform_remote_state.vpc.outputs.arvados_vpc_id
}

output "vpc_cidr" {
  value = data.terraform_remote_state.vpc.outputs.arvados_vpc_cidr
}

output "subnet_id" {
  value = data.terraform_remote_state.vpc.outputs.arvados_subnet_id
}

output "arvados_sg_id" {
  value = data.terraform_remote_state.vpc.outputs.arvados_sg_id
}

output "public_ip" {
  value = local.public_ip
}

output "private_ip" {
  value = local.private_ip
}

output "route53_dns_ns" {
  value = data.terraform_remote_state.vpc.outputs.route53_dns_ns
}

output "letsencrypt_iam_access_key_id" {
  value = data.terraform_remote_state.vpc.outputs.letsencrypt_iam_access_key_id
}

output "letsencrypt_iam_secret_access_key" {
  value = data.terraform_remote_state.vpc.outputs.letsencrypt_iam_secret_access_key
  sensitive = true
}

output "cluster_name" {
  value = data.terraform_remote_state.vpc.outputs.cluster_name
}

output "domain_name" {
  value = data.terraform_remote_state.vpc.outputs.domain_name
}

# Debian AMI's default user
output "deploy_user" {
  value = "admin"
}

output "region_name" {
  value = data.terraform_remote_state.vpc.outputs.region_name
}
