class CollectionNameOwnerUniqueOnlyNonExpired < ActiveRecord::Migration
  def up
    remove_index :collections, :name => 'collection_owner_uuid_name_unique'
    add_index(:collections, [:owner_uuid, :name], unique: true,
              where: 'expires_at is null',
              name: 'collection_owner_uuid_name_unique')
  end

  def down
    # it failed during up. is it going to pass now? should we do nothing?
    remove_index :collections, :name => 'collection_owner_uuid_name_unique'
    add_index(:collections, [:owner_uuid, :name], unique: true,
              name: 'collection_owner_uuid_name_unique')
  end
end
