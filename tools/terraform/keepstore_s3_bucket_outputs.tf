# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: Apache-2.0

output "keepstore_bucket_arn" {
  value = aws_s3_bucket.keepstore.*.arn
}
output "keepstore_bucket_id" {
  value = aws_s3_bucket.keepstore.*.id 
}
output "keepstore_bucket_policy_id" {
  value = aws_s3_bucket_policy.keepstore_bucket_policy.*.id
}
