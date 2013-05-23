class AddQsequenceToJobTasks < ActiveRecord::Migration
  def change
    add_column :job_tasks, :qsequence, :integer
  end
end
