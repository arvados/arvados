# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: CC-BY-SA-3.0

locals {
  allowed_ports = {
    http: "80",
    https: "443",
    ssh: "22",
  }
  availability_zone = data.aws_availability_zones.available.names[0]
  route53_public_zone = one(aws_route53_zone.public_zone[*])
  iam_user_letsencrypt = one(aws_iam_user.letsencrypt[*])
  iam_access_key_letsencrypt = one(aws_iam_access_key.letsencrypt[*])

  arvados_vpc_id = one(aws_vpc.arvados_vpc[*]) != null ? one(aws_vpc.arvados_vpc[*]).id : var.vpc_id
  arvados_vpc_cidr_block = one(aws_vpc.arvados_vpc[*])

  arvados_sg_id = one(aws_security_group.arvados_sg[*]) != null ? one(aws_security_group.arvados_sg[*]).id : var.sg_id

  private_subnet_id = one(aws_subnet.private_subnet[*]) != null ? one(aws_subnet.private_subnet[*]).id : var.private_subnet_id
  public_subnet_id = one(aws_subnet.public_subnet[*]) != null ? one(aws_subnet.public_subnet[*]).id : var.public_subnet_id

  public_hosts = var.private_only ? [] : var.user_facing_hosts
  private_hosts = concat(
    var.internal_service_hosts,
    var.private_only ? var.user_facing_hosts : []
  )
  public_ip = {
    for k, v in aws_eip.arvados_eip: k => v.public_ip
  }
  private_ip = var.private_ip
  cname_by_host = flatten([
    for host, aliases in var.dns_aliases : [
      for alias in aliases : {
        record = alias
        cname = host
      }
    ]
  ])
}
