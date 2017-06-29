# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

class AddStartFinishTimeToTasksAndPipelines < ActiveRecord::Migration
  def up
    add_column :job_tasks, :started_at, :datetime
    add_column :job_tasks, :finished_at, :datetime
    add_column :pipeline_instances, :started_at, :datetime
    add_column :pipeline_instances, :finished_at, :datetime
  end

  def down
    remove_column :job_tasks, :started_at
    remove_column :job_tasks, :finished_at
    remove_column :pipeline_instances, :started_at
    remove_column :pipeline_instances, :finished_at
  end
end
