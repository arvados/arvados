# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: Apache-2.0

provider "aws" {
  region  = var.aws_region
  profile = var.aws_profile
}

terraform {
  required_providers {
    aws = {
      version = "~> 3.31"
    }
  }
}
