class AddIsLockedByToJobs < ActiveRecord::Migration
  def change
    add_column :jobs, :is_locked_by, :string
  end
end
