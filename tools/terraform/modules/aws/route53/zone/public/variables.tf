# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: Apache-2.0

variable "zone_name"  {
  description = "Public zone to create"
  type        = string
}
variable "zone_config" {
  description = "Zone's config parameters"
  type    = map
  default = {}
}
variable "tags"  {
  type = map 
  default = {}
}
