class NoFilenamesInCollectionSearchIndex < ActiveRecord::Migration
  def up
    remove_index :collections, :name => 'collections_search_index'
    add_index :collections, ["owner_uuid", "modified_by_client_uuid", "modified_by_user_uuid", "portable_data_hash", "uuid", "name"], name: 'collections_search_index'
  end

  def down
    remove_index :collections, :name => 'collections_search_index'
    add_index :collections, ["owner_uuid", "modified_by_client_uuid", "modified_by_user_uuid", "portable_data_hash", "uuid", "name", "file_names"], name: 'collections_search_index'
  end
end
