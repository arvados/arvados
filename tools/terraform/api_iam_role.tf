# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: Apache-2.0

# Assume role for the instance
resource "aws_iam_role" "api_iam_role" {
    name = "${var.cluster}-api-iam-role"
    assume_role_policy = templatefile("${path.module}/iam_policy_assume_role.json", {})
}

# Associate the dispatcher policy to the role
resource "aws_iam_role_policy_attachment" "api_dispatcher_policies_attachment" {
    role       = aws_iam_role.api_iam_role.name
    policy_arn = aws_iam_policy.dispatcher_iam_policy.arn
}

# Associate letsencrypt modification policy to the role
resource "aws_iam_role_policy_attachment" "api_letsencrypt_route53_policies_attachment" {
    role       = aws_iam_role.api_iam_role.name
    policy_arn = aws_iam_policy.letsencrypt_route53_iam_policy.arn
}

# Add the role to the instance profile
resource "aws_iam_instance_profile" "api_instance_profile" {
  name  = "api_instance_profile"
  role = "${var.cluster}-api-iam-role"
}
