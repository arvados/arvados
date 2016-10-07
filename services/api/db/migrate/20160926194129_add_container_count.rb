class AddContainerCount < ActiveRecord::Migration
  def up
    add_column :container_requests, :container_count, :int, :default => 0
  end

  def down
    remove_column :container_requests, :container_count
  end
end
