# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: CC-BY-SA-3.0

terraform {
  required_version = "~> 1.3.0"
  required_providers {
    aws = {
      source = "hashicorp/aws"
      version = "~> 4.38.0"
    }
  }
}

provider "aws" {
  region = var.region_name
  default_tags {
    tags = merge(var.custom_tags, {
      Arvados = var.cluster_name
      Terraform = true
    })
  }
}

resource "aws_vpc" "arvados_vpc" {
  count = var.vpc_id == "" ? 1 : 0
  cidr_block = "10.1.0.0/16"
  enable_dns_hostnames = true
  enable_dns_support = true

  lifecycle {
    precondition {
      condition = (var.sg_id == "")
      error_message = "vpc_id should be set if sg_id is also set"
    }
  }
}
resource "aws_subnet" "public_subnet" {
  count = var.public_subnet_id == "" ? 1 : 0
  vpc_id = local.arvados_vpc_id
  availability_zone = local.availability_zone
  cidr_block = "10.1.1.0/24"

  lifecycle {
    precondition {
      condition = (var.vpc_id == "")
      error_message = "public_subnet_id should be set if vpc_id is also set"
    }
  }
}
resource "aws_subnet" "private_subnet" {
  count = var.private_subnet_id == "" ? 1 : 0
  vpc_id = local.arvados_vpc_id
  availability_zone = local.availability_zone
  cidr_block = "10.1.2.0/24"

  lifecycle {
    precondition {
      condition = (var.vpc_id == "")
      error_message = "private_subnet_id should be set if vpc_id is also set"
    }
  }
}

#
# Additional subnet on a different AZ is required if RDS is enabled
#
resource "aws_subnet" "additional_rds_subnet" {
  count = (var.additional_rds_subnet_id == "" && local.use_rds) ? 1 : 0
  vpc_id = local.arvados_vpc_id
  availability_zone = data.aws_availability_zones.available.names[1]
  cidr_block = "10.1.3.0/24"

  lifecycle {
    precondition {
      condition = (var.vpc_id == "")
      error_message = "additional_rds_subnet_id should be set if vpc_id is also set"
    }
  }
}

#
# VPC S3 access
#
resource "aws_vpc_endpoint" "s3" {
  count = var.vpc_id == "" ? 1 : 0
  vpc_id = local.arvados_vpc_id
  service_name = "com.amazonaws.${var.region_name}.s3"
}
resource "aws_vpc_endpoint_route_table_association" "compute_s3_route" {
  count = var.private_subnet_id == "" ? 1 : 0
  vpc_endpoint_id = aws_vpc_endpoint.s3[0].id
  route_table_id = aws_route_table.private_subnet_rt[0].id
}

#
# Internet access for Public IP instances
#
resource "aws_internet_gateway" "internet_gw" {
  count = var.vpc_id == "" ? 1 : 0
  vpc_id = local.arvados_vpc_id
}
resource "aws_eip" "arvados_eip" {
  for_each = toset(local.public_hosts)
  depends_on = [
    aws_internet_gateway.internet_gw
  ]
}
resource "aws_route_table" "public_subnet_rt" {
  count = var.public_subnet_id == "" ? 1 : 0
  vpc_id = local.arvados_vpc_id
  route {
    cidr_block = "0.0.0.0/0"
    gateway_id = aws_internet_gateway.internet_gw[0].id
  }
}
resource "aws_route_table_association" "public_subnet_assoc" {
  count = var.public_subnet_id == "" ? 1 : 0
  subnet_id = aws_subnet.public_subnet[0].id
  route_table_id = aws_route_table.public_subnet_rt[0].id
}

#
# Internet access for Private IP instances
#
resource "aws_eip" "nat_gw_eip" {
  count = var.private_subnet_id == "" ? 1 : 0
  depends_on = [
    aws_internet_gateway.internet_gw[0]
  ]
}
resource "aws_nat_gateway" "nat_gw" {
  count = var.private_subnet_id == "" ? 1 : 0
  # A NAT gateway should be placed on a subnet with an internet gateway
  subnet_id = aws_subnet.public_subnet[0].id
  allocation_id = aws_eip.nat_gw_eip[0].id
}
resource "aws_route_table" "private_subnet_rt" {
  count = var.private_subnet_id == "" ? 1 : 0
  vpc_id = local.arvados_vpc_id
  route {
    cidr_block = "0.0.0.0/0"
    nat_gateway_id = aws_nat_gateway.nat_gw[0].id
  }
}
resource "aws_route_table_association" "private_subnet_assoc" {
  count = var.private_subnet_id == "" ? 1 : 0
  subnet_id = aws_subnet.private_subnet[0].id
  route_table_id = aws_route_table.private_subnet_rt[0].id
}

resource "aws_security_group" "arvados_sg" {
  name = "arvados_sg"
  count = var.sg_id == "" ? 1 : 0
  vpc_id = aws_vpc.arvados_vpc[0].id

  lifecycle {
    precondition {
      condition = (var.vpc_id == "")
      error_message = "sg_id should be set if vpc_id is set"
    }
  }

  dynamic "ingress" {
    for_each = local.allowed_ports
    content {
      description = "Ingress rule for ${ingress.key}"
      from_port = "${ingress.value}"
      to_port = "${ingress.value}"
      protocol = "tcp"
      cidr_blocks = ["0.0.0.0/0"]
      ipv6_cidr_blocks = ["::/0"]
    }
  }
  # Allows communication between nodes in the VPC
  ingress {
    from_port = 0
    to_port = 0
    protocol = "-1"
    cidr_blocks = [ aws_vpc.arvados_vpc[0].cidr_block ]
  }
  # Even though AWS auto-creates an "allow all" egress rule,
  # Terraform deletes it, so we add it explicitly.
  egress {
    from_port = 0
    to_port = 0
    protocol = "-1"
    cidr_blocks = ["0.0.0.0/0"]
    ipv6_cidr_blocks = ["::/0"]
  }
}

#
# Route53 split-horizon DNS zones
#

# PUBLIC DNS
resource "aws_route53_zone" "public_zone" {
  count = var.private_only ? 0 : 1
  name = var.domain_name
}
resource "aws_route53_record" "public_a_record" {
  zone_id = try(local.route53_public_zone.id, "")
  for_each = local.public_ip
  name = each.key
  type = "A"
  ttl = 300
  records = [ each.value ]
}
resource "aws_route53_record" "main_a_record" {
  count = var.private_only ? 0 : 1
  zone_id = try(local.route53_public_zone.id, "")
  name = ""
  type = "A"
  ttl = 300
  records = [ local.public_ip["controller"] ]
}
resource "aws_route53_record" "public_cname_record" {
  zone_id = try(local.route53_public_zone.id, "")
  for_each = {
    for i in local.cname_by_host: i.record =>
      "${i.cname}.${var.domain_name}"
    if var.private_only == false
  }
  name = each.key
  type = "CNAME"
  ttl = 300
  records = [ each.value ]
}

# PRIVATE DNS
resource "aws_route53_zone" "private_zone" {
  name = var.domain_name
  vpc {
    vpc_id = local.arvados_vpc_id
  }
}
resource "aws_route53_record" "private_a_record" {
  zone_id = aws_route53_zone.private_zone.id
  for_each = local.private_ip
  name = each.key
  type = "A"
  ttl = 300
  records = [ each.value ]
}
resource "aws_route53_record" "private_main_a_record" {
  zone_id = aws_route53_zone.private_zone.id
  name = ""
  type = "A"
  ttl = 300
  records = [ local.private_ip["controller"] ]
}
resource "aws_route53_record" "private_cname_record" {
  zone_id = aws_route53_zone.private_zone.id
  for_each = {for i in local.cname_by_host: i.record => "${i.cname}.${var.domain_name}" }
  name = each.key
  type = "CNAME"
  ttl = 300
  records = [ each.value ]
}

#
# Route53's credentials for Let's Encrypt
#
resource "aws_iam_user" "letsencrypt" {
  count = var.private_only ? 0 : 1
  name = "${var.cluster_name}-letsencrypt"
  path = "/"
}

resource "aws_iam_access_key" "letsencrypt" {
  count = var.private_only ? 0 : 1
  user = local.iam_user_letsencrypt.name
}
resource "aws_iam_user_policy" "letsencrypt_iam_policy" {
  count = var.private_only ? 0 : 1
  name = "${var.cluster_name}-letsencrypt_iam_policy"
  user = local.iam_user_letsencrypt.name
  policy = jsonencode({
    "Version": "2012-10-17",
    "Statement": [{
      "Effect": "Allow",
      "Action": [
        "route53:ListHostedZones",
        "route53:GetChange"
      ],
      "Resource": [
          "*"
      ]
    },{
      "Effect" : "Allow",
      "Action" : [
        "route53:ChangeResourceRecordSets"
      ],
      "Resource" : [
        "arn:aws:route53:::hostedzone/${local.route53_public_zone.id}"
      ]
    }]
  })
}

