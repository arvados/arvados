class AddPropertiesToNode < ActiveRecord::Migration
  def up
    add_column :nodes, :properties, :text
  end

  def down
    remove_column :nodes, :properties
  end
end
