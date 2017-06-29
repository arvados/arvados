# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

class CreateAuthorizedKeys < ActiveRecord::Migration
  def change
    create_table :authorized_keys do |t|
      t.string :uuid, :null => false
      t.string :owner, :null => false
      t.string :modified_by_client
      t.string :modified_by_user
      t.datetime :modified_at
      t.string :name
      t.string :key_type
      t.string :authorized_user
      t.text :public_key
      t.datetime :expires_at

      t.timestamps
    end
    add_index :authorized_keys, :uuid, :unique => true
    add_index :authorized_keys, [:authorized_user, :expires_at]
  end
end
