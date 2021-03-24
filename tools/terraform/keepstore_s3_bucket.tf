# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: Apache-2.0

resource "aws_s3_bucket" "keepstore" {
  count  = var.keepstore_count
  bucket = "${var.cluster}-nyw5e-${format("%016d", count.index)}-volume"
  acl    = "private"

  website {
    index_document = "index.html"
    error_document = "error.html"
  }

  server_side_encryption_configuration {
    rule {
      apply_server_side_encryption_by_default {
        sse_algorithm = "AES256"
      }
    }
  }
  tags   = merge({"Name": "${var.cluster}-nyw5e-${format("%016d", count.index)}-bucket"},
                 local.resource_tags)
}

resource "aws_s3_bucket_policy" "keepstore_bucket_policy" {
  count  = var.keepstore_count
  bucket = aws_s3_bucket.keepstore.*.id[count.index]

  policy = templatefile("${path.module}/keepstore_iam_policy_s3_bucket.json", {
    id               = "${var.cluster}-nyw5e-${format("%016d", count.index)}-volume-policy",
    bucket_arn       = aws_s3_bucket.keepstore.*.arn[count.index]
    access_arns      = [aws_iam_role.keepstore_iam_assume_role.*.arn[count.index]]
  })
}
