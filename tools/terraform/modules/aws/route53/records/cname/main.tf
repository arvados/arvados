# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: Apache-2.0

resource "aws_route53_record" "cname_record" {
  for_each = var.zone_records_CNAME

  zone_id         = var.zone_id
  name            = lookup(each.value, "name", each.key)
  type            = "CNAME"
  ttl             = lookup(each.value, "ttl", 600)

  records         = each.value.records
  allow_overwrite = lookup(each.value, "allow_overwrite", null)
}
