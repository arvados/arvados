# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

class CreateKeepDisks < ActiveRecord::Migration
  def change
    create_table :keep_disks do |t|
      t.string :uuid, :null => false
      t.string :owner, :null => false
      t.string :modified_by_client
      t.string :modified_by_user
      t.datetime :modified_at
      t.string :ping_secret, :null => false
      t.string :node_uuid
      t.string :filesystem_uuid
      t.integer :bytes_total
      t.integer :bytes_free
      t.boolean :is_readable, :null => false, :default => true
      t.boolean :is_writable, :null => false, :default => true
      t.datetime :last_read_at
      t.datetime :last_write_at
      t.datetime :last_ping_at

      t.timestamps
    end
    add_index :keep_disks, :uuid, :unique => true
    add_index :keep_disks, :filesystem_uuid
    add_index :keep_disks, :node_uuid
    add_index :keep_disks, :last_ping_at
  end
end
