# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: CC-BY-SA-3.0

output "keepstore_iam_role_name" {
  value = aws_iam_role.keepstore_iam_role.name
}

output "compute_node_iam_role_name" {
  value = aws_iam_role.compute_node_iam_role.name
}

output "use_external_db" {
  value = var.use_external_db
}

output "loki_iam_policy_arn" {
  value = aws_iam_policy.loki_s3_full_access.arn
}
