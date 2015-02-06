class RenameReplicationAttributes < ActiveRecord::Migration
  RENAME = [[:redundancy, :replication_desired],
            [:redundancy_confirmed_as, :replication_confirmed],
            [:redundancy_confirmed_at, :replication_confirmed_at]]

  def up
    RENAME.each do |oldname, newname|
      rename_column :collections, oldname, newname
    end
    remove_column :collections, :redundancy_confirmed_by_client_uuid
    Collection.reset_column_information

    # Removing that column dropped some indexes. Let's put them back.
    add_index :collections, ["owner_uuid", "modified_by_client_uuid", "modified_by_user_uuid", "portable_data_hash", "uuid", "name", "file_names"], name: 'collections_search_index'
    execute "CREATE INDEX collections_full_text_search_idx ON collections USING gin(#{Collection.full_text_tsvector});"
  end

  def down
    remove_index :collections, name: 'collections_search_index'
    add_column :collections, :redundancy_confirmed_by_client_uuid, :string
    RENAME.reverse.each do |oldname, newname|
      rename_column :collections, newname, oldname
    end
    remove_index :collections, :name => 'collections_full_text_search_idx'
    Collection.reset_column_information

    execute "CREATE INDEX collections_full_text_search_idx ON collections USING gin(#{Collection.full_text_tsvector});"
    add_index :collections, ["owner_uuid", "modified_by_client_uuid", "modified_by_user_uuid", "portable_data_hash", "redundancy_confirmed_by_client_uuid", "uuid", "name", "file_names"], name: 'collections_search_index'
  end
end
