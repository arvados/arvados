# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

class RenameAuthorizedKeyAuthorizedUserToAuthorizedUserUuid < ActiveRecord::Migration
  def up
    remove_index :authorized_keys, [:authorized_user, :expires_at]
    rename_column :authorized_keys, :authorized_user, :authorized_user_uuid
    add_index :authorized_keys, [:authorized_user_uuid, :expires_at]
  end

  def down
    remove_index :authorized_keys, [:authorized_user_uuid, :expires_at]
    rename_column :authorized_keys, :authorized_user_uuid, :authorized_user
    add_index :authorized_keys, [:authorized_user, :expires_at]
  end
end
