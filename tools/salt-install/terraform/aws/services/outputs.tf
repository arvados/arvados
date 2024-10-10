# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: CC-BY-SA-3.0

output "vpc_id" {
  value = data.terraform_remote_state.vpc.outputs.arvados_vpc_id
}
output "cluster_int_cidr" {
  value = data.aws_vpc.arvados_vpc.cidr_block
}
output "arvados_subnet_id" {
  value = data.terraform_remote_state.vpc.outputs.public_subnet_id
}
output "compute_subnet_id" {
  value = data.terraform_remote_state.vpc.outputs.private_subnet_id
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
  value = var.deploy_user
}

output "region_name" {
  value = data.terraform_remote_state.vpc.outputs.region_name
}

output "ssl_password_secret_name" {
  value = aws_secretsmanager_secret.ssl_password_secret.name
}

output "database_address" {
  value = one(aws_db_instance.postgresql_service[*].address)
}

output "database_name" {
  value = one(aws_db_instance.postgresql_service[*].db_name)
}

output "database_username" {
  value = one(aws_db_instance.postgresql_service[*].username)
}

output "database_password" {
  value = one(aws_db_instance.postgresql_service[*].password)
  sensitive = true
}

output "database_version" {
  value = one(aws_db_instance.postgresql_service[*].engine_version_actual)
}

output "loki_iam_access_key_id" {
  value = data.terraform_remote_state.data-storage.outputs.loki_iam_access_key_id
}

output "loki_iam_secret_access_key" {
  value = data.terraform_remote_state.data-storage.outputs.loki_iam_secret_access_key
  sensitive = true
}
