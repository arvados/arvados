class RenameUserDefaultOwner < ActiveRecord::Migration
  def change
    rename_column :users, :default_owner, :default_owner_uuid
  end
end
