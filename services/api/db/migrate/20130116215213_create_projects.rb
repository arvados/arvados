# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

class CreateProjects < ActiveRecord::Migration
  def up
    create_table :projects do |t|
      t.string :uuid
      t.string :created_by_client
      t.string :created_by_user
      t.datetime :created_at
      t.string :modified_by_client
      t.string :modified_by_user
      t.datetime :modified_at
      t.string :name
      t.text :description

      t.timestamps
    end
    add_index :projects, :uuid, :unique => true
  end
  def down
    drop_table :projects
  end
end
