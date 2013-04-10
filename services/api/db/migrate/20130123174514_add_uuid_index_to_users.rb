class AddUuidIndexToUsers < ActiveRecord::Migration
  def change
    add_index :users, :uuid, :unique => true
  end
end
