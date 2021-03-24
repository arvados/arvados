# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: Apache-2.0

output "shell_iam_role_arn" {
  value = aws_iam_role.shell_iam_role.arn
}
output "shell_iam_role_id" {
  value = aws_iam_role.shell_iam_role.id
}
