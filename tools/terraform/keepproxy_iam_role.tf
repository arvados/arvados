# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: Apache-2.0

# Assume role for the instance
resource "aws_iam_role" "keepproxy_iam_role" {
    name = "${var.cluster}-keepproxy-iam-role"
    assume_role_policy = templatefile("${path.module}/iam_policy_assume_role.json", {})
}

# Associate letsencrypt modification policy to the role
resource "aws_iam_role_policy_attachment" "keepproxy_letsencrypt_route53_policies_attachment" {
    role       = aws_iam_role.keepproxy_iam_role.name
    policy_arn = aws_iam_policy.letsencrypt_route53_iam_policy.arn
}

# Add the role to the instance profile
resource "aws_iam_instance_profile" "keepproxy_instance_profile" {
  name  = "keepproxy_instance_profile"
  role = "${var.cluster}-keepproxy-iam-role"
}
