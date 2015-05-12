class ReadOnlyOnKeepServices < ActiveRecord::Migration
  def up
    add_column :keep_services, :read_only, :boolean, null: false, default: false
  end

  def down
    remove_column :keep_services, :read_only
  end
end
