# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: CC-BY-SA-3.0

locals {
  region_name = data.terraform_remote_state.vpc.outputs.region_name
  cluster_name = data.terraform_remote_state.vpc.outputs.cluster_name
}
