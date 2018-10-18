# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

class AddCurrentVersionUuidToCollectionSearchIndex < ActiveRecord::Migration
  disable_ddl_transaction!

  def up
    remove_index :collections, :name => 'collections_search_index'
    add_index :collections, ["owner_uuid", "modified_by_client_uuid", "modified_by_user_uuid", "portable_data_hash", "uuid", "name", "current_version_uuid"], name: 'collections_search_index', algorithm: :concurrently
  end

  def down
    remove_index :collections, :name => 'collections_search_index'
    add_index :collections, ["owner_uuid", "modified_by_client_uuid", "modified_by_user_uuid", "portable_data_hash", "uuid", "name"], name: 'collections_search_index', algorithm: :concurrently
  end
end
