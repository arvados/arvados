# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: Apache-2.0

# Assume role for the instance
resource "aws_iam_role" "keepstore_iam_assume_role" {
  count = var.keepstore_count
  name = "${var.cluster}-keepstore-${format("%02d", count.index)}-iam-role"
  assume_role_policy = file("${path.module}/iam_policy_assume_role.json")
}

# Associate the access bucket policy to the role
resource "aws_iam_role_policy_attachment" "keepstore_policies_attachment" {
  count = var.keepstore_count
  role       = aws_iam_role.keepstore_iam_assume_role.*.name[count.index]
  policy_arn = aws_iam_policy.keepstore_iam_policy.*.arn[count.index]
}

# Add the role to the instance profile
resource "aws_iam_instance_profile" "keepstore_instance_profile" {
  count = var.keepstore_count
  name  = "keepstore-${format("%02d", count.index)}_instance_profile"
  role = "${var.cluster}-keepstore-${format("%02d", count.index)}-iam-role"
}
