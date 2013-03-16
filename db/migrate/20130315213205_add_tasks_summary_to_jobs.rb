class AddTasksSummaryToJobs < ActiveRecord::Migration
  def change
    add_column :jobs, :tasks_summary, :text
  end
end
