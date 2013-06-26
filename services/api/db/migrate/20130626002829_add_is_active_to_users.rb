class AddIsActiveToUsers < ActiveRecord::Migration
  def change
    add_column :users, :is_active, :boolean, :default => false
  end
end
