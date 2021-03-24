# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: Apache-2.0

output "api_id" {
  value = module.api.id
}
output "api_private_dns_names" {
  value = aws_network_interface.api.private_dns_name
}
output "api_private_ip" {
  value = module.api.private_ip
}
output "api_private_eni_id" {
  value = aws_network_interface.api.id
}
output "api_public_ip" {
  value = aws_eip.cluster_api_public_ip.public_ip
}
