# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: CC-BY-SA-3.0

variable "use_external_db" {
  description = "Enable this if the database service won't be installed on these instances"
  type = bool
  default = false
}

variable "keep_cluster_data" {
  description = "Avoids state (database & keep blocks) to be destroyed. Needed for production clusters"
  type = bool
  default = false
}
