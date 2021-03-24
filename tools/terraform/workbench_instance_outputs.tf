# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: Apache-2.0

output "workbench_id" {
  value = module.workbench.id
}
output "workbench_private_dns_names" {
  value = module.workbench.private_dns
}
output "workbench_private_ip" {
  value = module.workbench.private_ip
}
output "workbench_private_eni_id" {
  value = aws_network_interface.workbench.id
}
output "workbench_public_ip" {
  value = aws_eip.cluster_workbench_public_ip.public_ip
}
