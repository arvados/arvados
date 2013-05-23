class AddCreatedByJobTaskToJobTasks < ActiveRecord::Migration
  def change
    add_column :job_tasks, :created_by_job_task, :string
  end
end
