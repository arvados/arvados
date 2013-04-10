class AddUuidToCollections < ActiveRecord::Migration
  def change
    add_column :collections, :uuid, :string
  end
end
