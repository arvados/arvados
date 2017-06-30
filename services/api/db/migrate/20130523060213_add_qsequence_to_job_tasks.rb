# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

class AddQsequenceToJobTasks < ActiveRecord::Migration
  def change
    add_column :job_tasks, :qsequence, :integer
  end
end
