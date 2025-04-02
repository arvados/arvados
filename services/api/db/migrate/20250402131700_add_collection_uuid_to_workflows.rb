# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

class AddCollectionUuidToWorkflows < ActiveRecord::Migration[7.1]

  def up
    remove_index :workflows, name: 'workflows_search_idx'
    add_column :workflows, :collection_uuid, :string, null: true
    add_index :workflows, ["uuid", "owner_uuid", "modified_by_client_uuid", "modified_by_user_uuid", "name", "collection_uuid"], name: 'workflows_search_index'
  end

  def down
    remove_index :workflows, name: 'workflows_search_index'
    remove_column :workflows, :collection_uuid
    add_index :workflows, ["uuid", "owner_uuid", "modified_by_client_uuid", "modified_by_user_uuid", "name"], name: 'workflows_search_idx'
  end

end
