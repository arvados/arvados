# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: Apache-2.0

## ZONE definition
module "r53_zone_public" {
  source      = "./modules/aws/route53/zone/public"
  zone_name   = var.r53_domain_name
  tags        = merge(
                      {"Name"    = var.r53_domain_name,
                       "Cluster" = var.cluster,
                      },
                      local.resource_tags,
                     )
}
module "r53_zone_private" {
  source      = "./modules/aws/route53/zone/private"
  zone_name   = var.r53_domain_name
  zone_config = {
    vpc_id = var.manage_vpc ? module.vpc.*.vpc_id[0] : var.vpc_id
  }
  tags        = merge(
                      {"Name"    = var.r53_domain_name,
                       "Cluster" = var.cluster,
                      },
                      local.resource_tags,
                     )
}
