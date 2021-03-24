# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: Apache-2.0

output "keepstore_id" {
  value = module.keepstore.*.id
}
output "keepstore_private_dns_names" {
  value = module.keepstore.*.private_dns
}
output "keepstore_private_ip" {
  value = module.keepstore.*.private_ip
}
output "keepstore_private_eni_id" {
  value = aws_network_interface.keepstore.*.id
}
