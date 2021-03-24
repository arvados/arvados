# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: Apache-2.0

module "shell" {
  source                 = "terraform-aws-modules/ec2-instance/aws"
  version                = "~> 2.17.0"

  name                   = "${var.cluster}-shell"
  instance_count         = 1

  iam_instance_profile   = "shell_instance_profile"
  ami                    = try(var.instance_ami["shell"], var.instance_ami["default"])
  instance_type          = try(var.instance_type["shell"], var.instance_type["default"])
  key_name               = var.key_name
  monitoring             = true

  tags                   = merge({"Name": "${var.cluster}-shell",
                                  "OsType": "LINUX"}, local.resource_tags)
  volume_tags            = merge({"Name": "${var.cluster}-shell"}, local.resource_tags)

  network_interface      = [{
    device_index = 0,
    network_interface_id = aws_network_interface.shell.id,
  }]

  # associate_public_ip_address = false
  ebs_optimized               = true
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
    volume_size           = try(var.data_bd_size["shell"], var.data_bd_size["default"])
    delete_on_termination = true,
    device_name           = "xvdh",
  }]
}

resource "aws_eip" "cluster_shell_public_ip" {
  vpc      = true
  instance = module.shell.id[0]
  network_interface = aws_network_interface.shell.id
  tags     = merge({"Name": "${var.cluster}-shell-ip"},local.resource_tags)
}

resource "aws_network_interface" "shell" {
  subnet_id       = var.manage_vpc ? module.vpc.0.public_subnets[0] : var.public_subnets_ids[0]
  # private_ips     = [cidrhost(var.vpc_subnet_cidrs[0], var.host_number["shell"])]
  security_groups = [
                     local.ssh_sg,
                     local.webshell_sg,
                    ]
  tags           = merge({"Name": "${var.cluster}-shell"}, local.resource_tags)
}

## Public A RRs
module "shell_route53_public_records_A" {
  source         = "./modules/aws/route53/records/a"
  zone_id        = module.r53_zone_public.id

  zone_records_A = {
    "shell" = {
      ttl     = "300",
      records = [aws_eip.cluster_shell_public_ip.public_ip]
    },
  }
}

## Private A RRs
module "shell_route53_private_records_A" {
  source         = "./modules/aws/route53/records/a"
  zone_id        = module.r53_zone_private.id

  zone_records_A = {
    "shell" = {
      ttl     = "300",
      records = aws_network_interface.shell.private_ips
    },
    "webshell" = {
      ttl     = "300",
      records = aws_network_interface.shell.private_ips
    },
  }
}
