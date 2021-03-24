# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: Apache-2.0

variable "zone_id"  {
  type = string
}
# Sadly, terraform can't iterate over maps of mixed values without lots
# of juggling with forced parameters in object definitions and
# other similarly nasty workarounds, so trying to keep it simple, we'll
# declare modules for each type of RRs, so code is simpler.
# More info here https://github.com/hashicorp/terraform/issues/19898
variable "zone_records_TXT" {
  description = "Map of TXT RRs to add to the zone to create"
  type        = map
  default     = {}
}
