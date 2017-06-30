# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

class RenameJobStepsToJobTasks < ActiveRecord::Migration
  def up
    rename_table :job_steps, :job_tasks
    rename_index :job_tasks, :index_job_steps_on_created_at, :index_job_tasks_on_created_at
    rename_index :job_tasks, :index_job_steps_on_job_uuid, :index_job_tasks_on_job_uuid
    rename_index :job_tasks, :index_job_steps_on_modified_at, :index_job_tasks_on_modified_at
    rename_index :job_tasks, :index_job_steps_on_sequence, :index_job_tasks_on_sequence
    rename_index :job_tasks, :index_job_steps_on_success, :index_job_tasks_on_success
    rename_index :job_tasks, :index_job_steps_on_uuid, :index_job_tasks_on_uuid
  end

  def down
    rename_index :job_steps, :index_job_tasks_on_created_at, :index_job_steps_on_created_at
    rename_index :job_steps, :index_job_tasks_on_job_uuid, :index_job_steps_on_job_uuid
    rename_index :job_steps, :index_job_tasks_on_modified_at, :index_job_steps_on_modified_at
    rename_index :job_steps, :index_job_tasks_on_sequence, :index_job_steps_on_sequence
    rename_index :job_steps, :index_job_tasks_on_success, :index_job_steps_on_success
    rename_index :job_steps, :index_job_tasks_on_uuid, :index_job_steps_on_uuid
    rename_table :job_tasks, :job_steps
  end
end
