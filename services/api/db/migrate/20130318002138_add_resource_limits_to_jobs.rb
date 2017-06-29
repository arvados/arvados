# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

class AddResourceLimitsToJobs < ActiveRecord::Migration
  def change
    add_column :jobs, :resource_limits, :text
  end
end
