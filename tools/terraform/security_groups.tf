# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: Apache-2.0

module "arvados_ssh_sg" {
  count               = var.manage_security_groups ? 1 : 0
  source              = "terraform-aws-modules/security-group/aws//modules/ssh"

  name                = "${var.cluster}_ssh_sg"
  description         = "SSH to Arvados VPC"
  vpc_id              = var.manage_vpc ? module.vpc.*.vpc_id[0] : var.vpc_id

  ingress_cidr_blocks = concat(
                               var.allowed_full_access_cidrs,
                               var.allowed_ssh_access_cidrs,
                               var.private_subnets,
                               var.public_subnets
                              )
  tags                = merge({"Name": "${var.cluster}-ssh-sg"},
                              local.resource_tags)
}
module "arvados_http_sg" {
  count               = var.manage_security_groups ? 1 : 0
  source              = "terraform-aws-modules/security-group/aws//modules/http-80"
  name                = "${var.cluster}_http_80_sg"
  description         = "HTTP security group"
  vpc_id              = var.manage_vpc ? module.vpc.*.vpc_id[0] : var.vpc_id
  ingress_cidr_blocks = concat(
                               var.allowed_full_access_cidrs,
                               var.allowed_http_access_cidrs,
                               var.private_subnets,
                               var.public_subnets
                              )
  tags                = merge({"Name": "${var.cluster}-http-sg"},
                              local.resource_tags)
}
module "arvados_https_sg" {
  count               = var.manage_security_groups ? 1 : 0
  source              = "terraform-aws-modules/security-group/aws//modules/https-443"
  name                = "${var.cluster}_https_443_sg"
  description         = "HTTPs security group"
  vpc_id              = var.manage_vpc ? module.vpc.*.vpc_id[0] : var.vpc_id
  ingress_cidr_blocks = concat(
                               var.allowed_full_access_cidrs,
                               var.allowed_http_access_cidrs,
                               var.private_subnets,
                               var.public_subnets
                              )
  tags                = merge({"Name": "${var.cluster}-https-sg"},
                               local.resource_tags)
}
module "arvados_webshell_sg" {
  count               = var.manage_security_groups ? 1 : 0
  source              = "terraform-aws-modules/security-group/aws"

  name                = "${var.cluster}_webshell_sg"
  description         = "Arvados access to webshell server"
  vpc_id              = var.manage_vpc ? module.vpc.*.vpc_id[0] : var.vpc_id

  ingress_cidr_blocks = concat(
                                var.allowed_full_access_cidrs,
                                var.allowed_ssh_access_cidrs,
                                var.private_subnets,
                        )
  ingress_with_cidr_blocks = [
    {
      from_port   = 4200
      to_port     = 4200
      protocol    = "tcp"
      description = "Webshell port"
      cidr_blocks = var.cluster_cidr
    },
  ]
  tags                = merge({"Name": "${var.cluster}-webshell-sg"},
                              local.resource_tags)
}
module "arvados_postgresql_sg" {
  count               = var.manage_security_groups ? 1 : 0
  source              = "terraform-aws-modules/security-group/aws//modules/postgresql"

  name                = "${var.cluster}_postgresql_sg"
  description         = "Arvados postgresql security group"
  vpc_id              = var.manage_vpc ? module.vpc.*.vpc_id[0] : var.vpc_id

  ingress_cidr_blocks = concat(
                               var.allowed_full_access_cidrs,
                               var.private_subnets,
                              )

  tags                = merge({"Name": "${var.cluster}-postgresql-sg"},
                              local.resource_tags)
}
module "arvados_keepstore_sg" {
  count               = var.manage_security_groups ? 1 : 0
  source = "terraform-aws-modules/security-group/aws"

  name        = "keepstore_sg"
  description = "Arvados security group for the keepstore service"
  vpc_id              = var.manage_vpc ? module.vpc.*.vpc_id[0] : var.vpc_id

  ingress_cidr_blocks      = [var.vpc_cidr]
  ingress_with_cidr_blocks = [
    {
      from_port   = 25107
      to_port     = 25107
      protocol    = "tcp"
      description = "Keepstore port"
      cidr_blocks = var.cluster_cidr
    },
  ]
  tags        = merge({"Name": "keepstore-sg"},local.resource_tags)
}
