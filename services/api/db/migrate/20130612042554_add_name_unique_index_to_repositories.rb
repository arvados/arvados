class AddNameUniqueIndexToRepositories < ActiveRecord::Migration
  def up
    remove_index :repositories, :name
    add_index :repositories, :name, :unique => true
  end

  def down
    remove_index :repositories, :name
    add_index :repositories, :name
  end
end
