# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: Apache-2.0

output "mx_records" {
  value = aws_route53_record.mx_record
}
