# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

class AddCreatedByJobTaskIndexToJobTasks < ActiveRecord::Migration
  def change
    add_index :job_tasks, :created_by_job_task_uuid
  end
end
