# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: Apache-2.0

resource "aws_key_pair" "keypair" {
  count      = var.enable_key_pair == true ? 1 : 0

  key_name   = var.key_name
  public_key = var.key_public_key == "" ? file(var.key_path) : var.key_public_key
  tags       = local.resource_tags
}
