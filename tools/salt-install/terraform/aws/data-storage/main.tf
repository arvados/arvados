# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: CC-BY-SA-3.0

terraform {
  required_version = "~> 1.3.0"
  required_providers {
    aws = {
      source = "hashicorp/aws"
      version = "~> 4.38.0"
    }
  }
}

provider "aws" {
  region = local.region_name
  default_tags {
    tags = merge(local.custom_tags, {
      Arvados = local.cluster_name
      Terraform = true
    })
  }
}

# S3 bucket and access resources for Keep blocks
resource "aws_s3_bucket" "keep_volume" {
  bucket = "${local.cluster_name}-nyw5e-000000000000000-volume"
}

resource "aws_iam_role" "keepstore_iam_role" {
  name = "${local.cluster_name}-keepstore-00-iam-role"
  assume_role_policy = "${file("../assumerolepolicy.json")}"
}

resource "aws_iam_role" "compute_node_iam_role" {
  name = "${local.cluster_name}-compute-node-00-iam-role"
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
  roles = [
    aws_iam_role.keepstore_iam_role.name,
    aws_iam_role.compute_node_iam_role.name,
  ]
  policy_arn = aws_iam_policy.s3_full_access.arn
}

# S3 bucket and access resources for Loki
resource "aws_s3_bucket" "loki_storage" {
  bucket = "${local.cluster_name}-loki-object-storage"
}

resource "aws_iam_user" "loki" {
  name = "${local.cluster_name}-loki"
  path = "/"
}

resource "aws_iam_access_key" "loki" {
  user = aws_iam_user.loki.name
}

resource "aws_iam_user_policy" "loki_s3_full_access" {
  name = "${local.cluster_name}_loki_s3_full_access"
  user = aws_iam_user.loki.name
  policy = jsonencode({
    Version: "2012-10-17",
    Id: "Loki S3 storage policy",
    Statement: [{
      Effect: "Allow",
      Action: [
        "s3:*",
      ],
      Resource: [
        "arn:aws:s3:::${local.cluster_name}-loki-object-storage",
        "arn:aws:s3:::${local.cluster_name}-loki-object-storage/*"
      ]
    }]
  })
}
