# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: Apache-2.0

module "database" {
  source                 = "terraform-aws-modules/ec2-instance/aws"
  version                = "~> 2.17.0"

  name                   = "${var.cluster}-database"
  instance_count         = 1

  ami                    = try(var.instance_ami["database"], var.instance_ami["default"])
  instance_type          = try(var.instance_type["database"], var.instance_type["default"])
  key_name               = var.key_name
  monitoring             = true

  tags                   = merge({"Name": "${var.cluster}-database",
                                  "OsType": "LINUX"}, local.resource_tags)
  volume_tags            = merge({"Name": "${var.cluster}-database"}, local.resource_tags)

  network_interface      = [{
    device_index = 0,
    network_interface_id = aws_network_interface.database.id,
  }]

  ebs_optimized          = true
  user_data               = templatefile("_user_data.sh", {})

  root_block_device           = [{
    encrypted             = true,
    kms_key_id            = var.kms_key_id,
    volume_size           = var.root_bd_size,
    delete_on_termination = true,
  }]
  ebs_block_device            = [{
    encrypted             = true,
    kms_key_id            = var.kms_key_id,
    volume_size           = try(var.data_bd_size["database"], var.data_bd_size["default"])
    delete_on_termination = true,
    device_name           = "xvdh",
  }]
}

resource "aws_network_interface" "database" {
  subnet_id       = var.manage_vpc ? module.vpc.0.private_subnets[0] : var.private_subnets_ids[0]
  # private_ips     = [cidrhost(var.vpc_subnet_cidrs[0], var.host_number["database"])]
  security_groups = [
                     local.ssh_sg,
                     local.postgresql_sg,
                    ]
  tags            = merge({"Name": "${var.cluster}-database"}, local.resource_tags)
}

## Private A RRs
module "database_route53_private_records_A" {
  source         = "./modules/aws/route53/records/a"
  zone_id        = module.r53_zone_private.id

  zone_records_A = {
    "database" = {
      ttl     = "300",
      records = aws_network_interface.database.private_ips

    },
    "db" = {
      ttl     = "300",
      records = aws_network_interface.database.private_ips
    }
  }
}
