# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: Apache-2.0

module "keepstore" {
  count                = var.keepstore_count
  source               = "terraform-aws-modules/ec2-instance/aws"
  version              = "~> 2.17.0"

  name                 = "${var.cluster}-keepstore-${format("%02d", count.index)}"
  instance_count       = 1

  ami                  = try(var.instance_ami["keepstore"], var.instance_ami["default"])
  instance_type        = try(var.instance_type["keepstore"], var.instance_type["default"])

  iam_instance_profile = "keepstore-${format("%02d", count.index)}_instance_profile"
  key_name             = var.key_name
  monitoring           = true

  tags                 = merge({"Name": "${var.cluster}-keepstore-${format("%02d", count.index)}",
                               "OsType": "LINUX"}, local.resource_tags)
  volume_tags          = merge({"Name": "${var.cluster}-keepstore-${format("%02d", count.index)}"},
                               local.resource_tags)

  network_interface    = [{
    device_index         = 0,
    network_interface_id = aws_network_interface.keepstore.*.id[count.index],
  }]

  # associate_public_ip_address = false
  ebs_optimized        = true
  user_data               = templatefile("_user_data.sh", {})

  root_block_device    = [{
    encrypted             = true,
    kms_key_id            = var.kms_key_id,
    volume_size           = var.root_bd_size,
    delete_on_termination = true,
  }]
  ebs_block_device     = [{
    encrypted             = true,
    kms_key_id            = var.kms_key_id,
    volume_size           = try(var.data_bd_size["keepstore"], var.data_bd_size["default"])
    delete_on_termination = true,
    device_name           = "xvdh",
  }]
}

# resource "aws_eip" "cluster_keepstore_public_ip" {
#   count     = var.keepstore_count
#   vpc      = true
#   instance = module.keepstore.*.id[count.index][0]
#   network_interface = aws_network_interface.keepstore.*.id[count.index]
#   tags     = merge({"Name": "${var.cluster}-keepstore-${format("%02d", count.index)}-ip"},local.resource_tags)
# }

resource "aws_network_interface" "keepstore" {
  count            = var.keepstore_count
  subnet_id        = var.manage_vpc ? module.vpc.0.private_subnets[0] : var.private_subnets_ids[0]
  # private_ips     = [cidrhost(var.vpc_subnet_cidrs[0], var.host_number["keepstore"])]
  security_groups = [
                     local.ssh_sg,
                     local.keepstore_sg,
                    ]
  tags      = merge({"Name": "${var.cluster}-keepstore-${format("%02d", count.index)}"},
                    local.resource_tags)
}

### FIXME! Needs improvement
## Private A RRs
module "keepstore_route53_private_records_A" {
  source         = "./modules/aws/route53/records/a"
  zone_id        = module.r53_zone_private.id

  zone_records_A = {
    "keep0" = {
      ttl     = "300",
      records = aws_network_interface.keepstore.0.private_ips
    },
    "keep1" = {
      ttl     = "300",
      records = aws_network_interface.keepstore.1.private_ips
    },
  }
}
