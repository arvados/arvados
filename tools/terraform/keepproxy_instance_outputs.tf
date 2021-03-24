# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: Apache-2.0

output "keepproxy_id" {
  value = module.keepproxy.id
}
output "keepproxy_private_dns_names" {
  value = module.keepproxy.private_dns
}
output "keepproxy_private_ip" {
  value = module.keepproxy.private_ip
}
output "keepproxy_private_eni_id" {
  value = aws_network_interface.keepproxy.id
}
output "keepproxy_public_ip" {
  value = aws_eip.cluster_keepproxy_public_ip.public_ip
}
