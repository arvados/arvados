# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

class RenameResourceLimitsToRuntimeConstraints < ActiveRecord::Migration
  def change
    rename_column :jobs, :resource_limits, :runtime_constraints
  end
end
