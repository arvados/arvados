class AddContainerLockCount < ActiveRecord::Migration
  def change
    add_column :containers, :lock_count, :int, :null => false, :default => 0
  end
end
