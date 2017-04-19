class AddCreatedByJobTaskIndexToJobTasks < ActiveRecord::Migration
  def change
    add_index :job_tasks, :created_by_job_task_uuid
  end
end
