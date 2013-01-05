class RenameCollectionCreatedByClient < ActiveRecord::Migration
  def up
    rename_column :collections, :create_by_client, :created_by_client
  end

  def down
    rename_column :collections, :created_by_client, :create_by_client
  end
end
