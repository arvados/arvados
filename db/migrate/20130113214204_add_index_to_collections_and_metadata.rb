class AddIndexToCollectionsAndMetadata < ActiveRecord::Migration
  def up
    add_index :collections, :uuid, :unique => true
    add_index :metadata, :uuid, :unique => true
  end
  def down
    remove_index :metadata, :uuid
    remove_index :collections, :uuid
  end
end
