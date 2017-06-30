# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

class AddIsLockedByToJobs < ActiveRecord::Migration
  def change
    add_column :jobs, :is_locked_by, :string
  end
end
