# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: Apache-2.0

output "keypair_name" {
  description = "Name of the SSH keypair applied to the instances."
  value       = var.key_name
}
