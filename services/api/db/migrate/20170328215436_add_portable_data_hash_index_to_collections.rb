class AddPortableDataHashIndexToCollections < ActiveRecord::Migration
  def change
    add_index :collections, :portable_data_hash
  end
end
