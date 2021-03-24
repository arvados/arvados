# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: Apache-2.0

output "arvados_ssh_security_group_id" {
  description = "The ID of the arvados SSH security group"
  value       = var.manage_security_groups ? module.arvados_ssh_sg.0.this_security_group_id : var.vpc_security_group_ids["ssh"]
}
output "arvados_webshell_security_group_id" {
  description = "The ID of the arvados Webshell security group"
  value       = var.manage_security_groups ? module.arvados_webshell_sg.0.this_security_group_id : var.vpc_security_group_ids["webshell"]
}
output "arvados_http_security_group_id" {
  description = "The ID of the arvados HTTP security group"
  value       = var.manage_security_groups ? module.arvados_http_sg.0.this_security_group_id : var.vpc_security_group_ids["http"]
}
output "arvados_https_security_group_id" {
  description = "The ID of the arvados HTTPS security group"
  value       = var.manage_security_groups ? module.arvados_https_sg.0.this_security_group_id : var.vpc_security_group_ids["https"]
}
output "arvados_postgresql_security_group_id" {
  description = "The ID of the arvados Postgresql security group"
  value       = var.manage_security_groups ? module.arvados_postgresql_sg.0.this_security_group_id : var.vpc_security_group_ids["postgresql"]
}
output "arvados_keepstore_security_group_id" {
  description = "The ID of the arvados Keepstore security group"
  value       = var.manage_security_groups ? module.arvados_keepstore_sg.0.this_security_group_id : var.vpc_security_group_ids["keepstore"]
}
