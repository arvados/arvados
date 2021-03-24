# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: Apache-2.0

output "keepstore_iam_policy_id" {
  value = aws_iam_policy.keepstore_iam_policy.*.id
}
output "keepstore_iam_policy_arn" {
  value = aws_iam_policy.keepstore_iam_policy.*.arn
}
output "keepstore_iam_assume_role_arn" {
  value = aws_iam_role.keepstore_iam_assume_role.*.arn
}
output "keepstore_iam_assume_role_id" {
  value = aws_iam_role.keepstore_iam_assume_role.*.id
}
