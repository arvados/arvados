class CollectionFileNames < ActiveRecord::Migration
  def change
    add_column :collections, :file_names, :string, :limit => 2**16
  end
end
