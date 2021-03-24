# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: Apache-2.0

output "id" {
  value = aws_route53_zone.this.id
}
output "name" {
  value = aws_route53_zone.this.name
}
output "name_servers" {
  value = aws_route53_zone.this.name_servers
}
