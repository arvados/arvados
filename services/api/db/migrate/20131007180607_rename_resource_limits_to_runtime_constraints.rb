class RenameResourceLimitsToRuntimeConstraints < ActiveRecord::Migration
  def change
    rename_column :jobs, :resource_limits, :runtime_constraints
  end
end
