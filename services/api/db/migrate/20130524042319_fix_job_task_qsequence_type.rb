# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

class FixJobTaskQsequenceType < ActiveRecord::Migration
  def up
    change_column :job_tasks, :qsequence, :integer, :limit => 8
  end

  def down
    change_column :job_tasks, :qsequence, :integer
  end
end
