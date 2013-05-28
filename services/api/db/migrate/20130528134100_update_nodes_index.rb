class UpdateNodesIndex < ActiveRecord::Migration
  def up
    remove_index :nodes, :hostname
    add_index :nodes, :hostname
  end
  def down
    remove_index :nodes, :hostname
    add_index :nodes, :hostname, :unique => true
  end
end
