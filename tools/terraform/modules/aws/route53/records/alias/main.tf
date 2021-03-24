# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: Apache-2.0

resource "aws_route53_record" "alias_record" {
  for_each = var.zone_records_ALIAS

  zone_id         = var.zone_id
  name            = lookup(each.value, "name", each.key)
  type            = lookup(each.value, "type", "A")

  alias {
    name                   = each.value.alias.name
    zone_id                = each.value.alias.zone_id
    evaluate_target_health = lookup(each.value.alias, "evaluate_target_health", false)
  }
  allow_overwrite = lookup(each.value, "allow_overwrite", null)
}
