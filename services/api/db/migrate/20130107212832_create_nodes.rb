# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

class CreateNodes < ActiveRecord::Migration
  def up
    create_table :nodes do |t|
      t.string :uuid
      t.string :created_by_client
      t.string :created_by_user
      t.datetime :created_at
      t.string :modified_by_client
      t.string :modified_by_user
      t.datetime :modified_at
      t.integer :slot_number
      t.string :hostname
      t.string :domain
      t.string :ip_address
      t.datetime :first_ping_at
      t.datetime :last_ping_at
      t.text :info

      t.timestamps
    end
    add_index :nodes, :uuid, :unique => true
    add_index :nodes, :slot_number, :unique => true
    add_index :nodes, :hostname, :unique => true
  end
  def down
    drop_table :nodes
  end
end
