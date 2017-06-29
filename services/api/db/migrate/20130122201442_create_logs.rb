# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

class CreateLogs < ActiveRecord::Migration
  def up
    create_table :logs do |t|
      t.string :uuid
      t.string :created_by_client
      t.string :created_by_user
      t.string :modified_by_client
      t.string :modified_by_user
      t.string :object_kind
      t.string :object_uuid
      t.datetime :event_at
      t.string :event_type
      t.text :summary
      t.text :info

      t.timestamps
    end
    add_index :logs, :uuid, :unique => true
    add_index :logs, :object_kind
    add_index :logs, :object_uuid
    add_index :logs, :event_type
    add_index :logs, :event_at
    add_index :logs, :summary
  end

  def down
    drop_table :logs  end
end
