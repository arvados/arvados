class CollectionNameOwnerUniqueOnlyNonExpired < ActiveRecord::Migration
  def find_index
    indexes = ActiveRecord::Base.connection.indexes('collections')
    name_owner_index = indexes.select do |index|
      index.name == 'collection_owner_uuid_name_unique'
    end
    name_owner_index
  end

  def up
    remove_index :collections, :name => 'collection_owner_uuid_name_unique' if !find_index.empty?
    add_index(:collections, [:owner_uuid, :name], unique: true,
              where: 'expires_at is null',
              name: 'collection_owner_uuid_name_unique')
  end

  def down
    # it failed during up. is it going to pass now? should we do nothing?
    remove_index :collections, :name => 'collection_owner_uuid_name_unique' if !find_index.empty?
    add_index(:collections, [:owner_uuid, :name], unique: true,
              name: 'collection_owner_uuid_name_unique')
  end
end
