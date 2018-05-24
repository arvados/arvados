# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

class AddRedirectToUserUuidToUsers < ActiveRecord::Migration
  def up
    add_column :users, :redirect_to_user_uuid, :string
    User.reset_column_information
    remove_index :users, name: 'users_search_index'
    add_index :users, User.searchable_columns('ilike') - ['prefs'], name: 'users_search_index'
  end

  def down
    remove_index :users, name: 'users_search_index'
    remove_column :users, :redirect_to_user_uuid
    User.reset_column_information
    add_index :users, User.searchable_columns('ilike') - ['prefs'], name: 'users_search_index'
  end
end
