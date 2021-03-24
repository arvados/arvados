# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: Apache-2.0

output "vpc_route53_private_zone_id" {
  value = module.r53_zone_private.id
}
output "vpc_route53_private_zone_name" {
  value = module.r53_zone_private.name
}
output "vpc_route53_private_name_servers" {
  value = module.r53_zone_private.name_servers
}
output "vpc_route53_public_zone_id" {
  value = module.r53_zone_public.id
}
output "vpc_route53_public_zone_name" {
  value = module.r53_zone_public.name
}
output "vpc_route53_public_name_servers" {
  value = module.r53_zone_public.name_servers
}
