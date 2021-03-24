# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: Apache-2.0

resource "aws_iam_policy" "dispatcher_iam_policy" {
  name  = "${var.cluster}-dispatcher-iam-role-policy"
  description = "Policy to allow API to launch compute instances"
  policy = templatefile("${path.module}/dispatcher_iam_policy.json", {
    "cluster" = var.cluster
  })
}
