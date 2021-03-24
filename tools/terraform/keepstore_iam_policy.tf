# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: Apache-2.0

# IAM policy to access the bucket
resource "aws_iam_policy" "keepstore_iam_policy" {
  count = var.keepstore_count
  name  = "${var.cluster}-keepstore-${format("%02d", count.index)}-iam-role-policy"
  description = "Policy to allow writing to the S3 bucket ${var.cluster}-nyw5e-${format("%016d", count.index)}-volume-policy"
  policy = templatefile("${path.module}/keepstore_iam_policy.json", {
    "bucket_arn" = aws_s3_bucket.keepstore.*.arn[count.index]
  })
}
