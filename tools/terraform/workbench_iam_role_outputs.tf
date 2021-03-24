# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: Apache-2.0

output "workbench_iam_role_arn" {
  value = aws_iam_role.workbench_iam_role.arn
}
output "workbench_iam_role_id" {
  value = aws_iam_role.workbench_iam_role.id
}
