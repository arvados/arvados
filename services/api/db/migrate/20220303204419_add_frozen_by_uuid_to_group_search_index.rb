# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

class AddFrozenByUuidToGroupSearchIndex < ActiveRecord::Migration[5.0]
  disable_ddl_transaction!

  def up
    remove_index :groups, :name => 'groups_search_index'
    add_index :groups, ["uuid", "owner_uuid", "modified_by_client_uuid", "modified_by_user_uuid", "name", "group_class", "frozen_by_uuid"], name: 'groups_search_index', algorithm: :concurrently
  end

  def down
    remove_index :groups, :name => 'groups_search_index'
    add_index :groups, ["uuid", "owner_uuid", "modified_by_client_uuid", "modified_by_user_uuid", "name", "group_class"], name: 'groups_search_index', algorithm: :concurrently
  end
end
