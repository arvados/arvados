class AddLogToJobs < ActiveRecord::Migration
  def change
    add_column :jobs, :log, :string
  end
end
