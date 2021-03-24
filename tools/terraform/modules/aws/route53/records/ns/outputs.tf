# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: Apache-2.0

output "ns_records" {
  value = aws_route53_record.ns_record
}
