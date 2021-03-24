# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: Apache-2.0

resource "aws_iam_policy" "letsencrypt_route53_iam_policy" {
  name  = "${var.cluster}-letsencrypt_route53-iam-role-policy"
  description = "Policy to allow API to add records to the public zone"
  policy = templatefile("${path.module}/letsencrypt_route53_iam_policy.json", {
    "cluster" = var.cluster
    "zone_id" = module.r53_zone_public.id
  })
}
