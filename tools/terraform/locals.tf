locals {
  resource_tags = merge(
                        { "Terraform" = "true" },
                        { "Environment" = var.environment },
                        { "Namespace" = var.namespace },
                        { "Cluster" = var.cluster },
                        var.tags,
                       )
  ssh_sg         = var.manage_security_groups ? module.arvados_ssh_sg.0.this_security_group_id : try(var.vpc_security_group_ids["ssh"], var.vpc_security_group_ids["default"])
  http_sg        = var.manage_security_groups ? module.arvados_http_sg.0.this_security_group_id : try(var.vpc_security_group_ids["http"], var.vpc_security_group_ids["default"])
  https_sg       = var.manage_security_groups ? module.arvados_https_sg.0.this_security_group_id : try(var.vpc_security_group_ids["https"], var.vpc_security_group_ids["default"])
  webshell_sg    = var.manage_security_groups ? module.arvados_webshell_sg.0.this_security_group_id : try(var.vpc_security_group_ids["webshell"], var.vpc_security_group_ids["default"])
  postgresql_sg  = var.manage_security_groups ? module.arvados_postgresql_sg.0.this_security_group_id : try(var.vpc_security_group_ids["postgresql"], var.vpc_security_group_ids["default"])
  keepstore_sg   = var.manage_security_groups ? module.arvados_keepstore_sg.0.this_security_group_id : try(var.vpc_security_group_ids["keepstore"], var.vpc_security_group_ids["default"])

}
# locals {
#   office_ipv4        = data.terraform_remote_state.globals.outputs.office_cidrs_ipv4
#   r53_description    = "Private zone for ${data.terraform_remote_state.globals.outputs.environment}"
#   r53_top_domain     = "arvadosapi.com"
#   r53_parent_zone_id = data.terraform_remote_state.parent_dns.outputs.zone_id
# }
