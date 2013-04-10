class AddDefaultOwnerToUsers < ActiveRecord::Migration
  def change
    add_column :users, :default_owner, :string
  end
end
