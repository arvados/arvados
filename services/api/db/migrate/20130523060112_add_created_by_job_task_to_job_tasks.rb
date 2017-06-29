# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

class AddCreatedByJobTaskToJobTasks < ActiveRecord::Migration
  def change
    add_column :job_tasks, :created_by_job_task, :string
  end
end
