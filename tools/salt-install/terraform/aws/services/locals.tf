# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: CC-BY-SA-3.0

locals {
  region_name = data.terraform_remote_state.vpc.outputs.region_name
  cluster_name = data.terraform_remote_state.vpc.outputs.cluster_name
  use_external_db = data.terraform_remote_state.data-storage.outputs.use_external_db
  private_only = data.terraform_remote_state.vpc.outputs.private_only
  public_ip = data.terraform_remote_state.vpc.outputs.public_ip
  private_ip = data.terraform_remote_state.vpc.outputs.private_ip
  pubkey_path = pathexpand(var.pubkey_path)
  public_hosts = data.terraform_remote_state.vpc.outputs.public_hosts
  private_hosts = data.terraform_remote_state.vpc.outputs.private_hosts
  user_facing_hosts = data.terraform_remote_state.vpc.outputs.user_facing_hosts
  internal_service_hosts = data.terraform_remote_state.vpc.outputs.internal_service_hosts
  ssl_password_secret_name = "${local.cluster_name}-${var.ssl_password_secret_name_suffix}"
  instance_ami_id = var.instance_ami != "" ? var.instance_ami : data.aws_ami.debian-11.image_id
  custom_tags = data.terraform_remote_state.vpc.outputs.custom_tags
  compute_node_iam_role_name = data.terraform_remote_state.data-storage.outputs.compute_node_iam_role_name
  instance_profile = {
    default = aws_iam_instance_profile.default_instance_profile
    workbench = aws_iam_instance_profile.dispatcher_instance_profile
    keep0 = aws_iam_instance_profile.keepstore_instance_profile
  }
  private_subnet_id = data.terraform_remote_state.vpc.outputs.private_subnet_id
  public_subnet_id = data.terraform_remote_state.vpc.outputs.public_subnet_id
  additional_rds_subnet_id = data.terraform_remote_state.vpc.outputs.additional_rds_subnet_id
  arvados_sg_id = data.terraform_remote_state.vpc.outputs.arvados_sg_id
  eip_id = data.terraform_remote_state.vpc.outputs.eip_id
  keepstore_iam_role_name = data.terraform_remote_state.data-storage.outputs.keepstore_iam_role_name
  use_rds = (var.use_rds && data.terraform_remote_state.vpc.outputs.use_rds)
  rds_username = var.rds_username != "" ? var.rds_username : "${local.cluster_name}_arvados"
  rds_password = var.rds_password != "" ? var.rds_password : one(random_string.default_rds_password[*].result)
  rds_allocated_storage = var.rds_allocated_storage
  rds_max_allocated_storage = max(var.rds_max_allocated_storage, var.rds_allocated_storage)
  rds_instance_type = var.rds_instance_type
  rds_backup_retention_period = var.rds_backup_retention_period
  rds_backup_before_deletion = var.rds_backup_before_deletion
  rds_final_backup_name = var.rds_final_backup_name != "" ? var.rds_final_backup_name : "arvados-${local.cluster_name}-db-final-snapshot"
  rds_postgresql_version = var.rds_postgresql_version
}
