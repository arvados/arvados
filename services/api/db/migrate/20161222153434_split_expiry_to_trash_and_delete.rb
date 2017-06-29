# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

class SplitExpiryToTrashAndDelete < ActiveRecord::Migration
  def up
    Collection.transaction do
      add_column(:collections, :trash_at, :datetime)
      add_index(:collections, :trash_at)
      add_column(:collections, :is_trashed, :boolean, null: false, default: false)
      add_index(:collections, :is_trashed)
      rename_column(:collections, :expires_at, :delete_at)
      add_index(:collections, :delete_at)

      Collection.reset_column_information
      Collection.
        where('delete_at is not null and delete_at <= statement_timestamp()').
        delete_all
      Collection.
        where('delete_at is not null').
        update_all('is_trashed = true, trash_at = statement_timestamp()')
      add_index(:collections, [:owner_uuid, :name],
                unique: true,
                where: 'is_trashed = false',
                name: 'index_collections_on_owner_uuid_and_name')
      remove_index(:collections,
                   name: 'collection_owner_uuid_name_unique')
    end
  end

  def down
    Collection.transaction do
      remove_index(:collections, :delete_at)
      rename_column(:collections, :delete_at, :expires_at)
      add_index(:collections, [:owner_uuid, :name],
                unique: true,
                where: 'expires_at is null',
                name: 'collection_owner_uuid_name_unique')
      remove_index(:collections,
                   name: 'index_collections_on_owner_uuid_and_name')
      remove_column(:collections, :is_trashed)
      remove_index(:collections, :trash_at)
      remove_column(:collections, :trash_at)
    end
  end
end
