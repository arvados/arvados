# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: Apache-2.0

output "shell_id" {
  value = module.shell.id
}
output "shell_private_dns_names" {
  value = module.shell.private_dns
}
output "shell_private_ip" {
  value = module.shell.private_ip
}
output "shell_private_eni_id" {
  value = aws_network_interface.shell.id
}
output "shell_public_ip" {
  value = aws_eip.cluster_shell_public_ip.public_ip
}
