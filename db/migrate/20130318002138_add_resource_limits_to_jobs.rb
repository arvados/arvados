class AddResourceLimitsToJobs < ActiveRecord::Migration
  def change
    add_column :jobs, :resource_limits, :text
  end
end
