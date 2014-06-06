class AddSuppliedScriptVersion < ActiveRecord::Migration
  def up
    add_column :jobs, :supplied_script_version, :string
  end

  def down
    remove_column :jobs, :supplied_script_version, :string
  end
end
