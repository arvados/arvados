# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: CC-BY-SA-3.0

terraform {
  required_providers {
    aws = {
      source = "hashicorp/aws"
    }
  }
}

provider "aws" {
  region = local.region_name
  default_tags {
    tags = {
      Arvados = local.cluster_name
    }
  }
}

# S3 bucket and access resources for Keep blocks
resource "aws_s3_bucket" "keep_volume" {
  bucket = "${local.cluster_name}-nyw5e-000000000000000-volume"
}

resource "aws_s3_bucket_acl" "keep_volume_acl" {
  bucket = aws_s3_bucket.keep_volume.id
  acl = "private"
}

# Avoid direct public access to Keep blocks
resource "aws_s3_bucket_public_access_block" "keep_volume_public_access" {
  bucket = aws_s3_bucket.keep_volume.id

  block_public_acls   = true
  block_public_policy = true
  ignore_public_acls  = true
}

resource "aws_iam_role" "keepstore_iam_role" {
  name = "${local.cluster_name}-keepstore-00-iam-role"
  assume_role_policy = "${file("../assumerolepolicy.json")}"
}

resource "aws_iam_policy" "s3_full_access" {
  name = "${local.cluster_name}_s3_full_access"
  policy = jsonencode({
    Version: "2012-10-17",
    Id: "arvados-keepstore policy",
    Statement: [{
      Effect: "Allow",
      Action: [
        "s3:*",
      ],
      Resource: [
        "arn:aws:s3:::${local.cluster_name}-nyw5e-000000000000000-volume",
        "arn:aws:s3:::${local.cluster_name}-nyw5e-000000000000000-volume/*"
      ]
    }]
  })
}

resource "aws_iam_policy_attachment" "s3_full_access_policy_attachment" {
  name = "${local.cluster_name}_s3_full_access_attachment"
  roles = [ aws_iam_role.keepstore_iam_role.name ]
  policy_arn = aws_iam_policy.s3_full_access.arn
}

