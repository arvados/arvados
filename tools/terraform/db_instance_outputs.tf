# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: Apache-2.0

output "database_id" {
  value = module.database.id
}
output "database_private_dns_names" {
  value = module.database.private_dns
}
output "database_private_ip" {
  value = module.database.private_ip
}
output "database_private_eni_id" {
  value = aws_network_interface.database.id
}
