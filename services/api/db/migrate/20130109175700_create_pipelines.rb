# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

class CreatePipelines < ActiveRecord::Migration
  def up
    create_table :pipelines do |t|
      t.string :uuid
      t.string :created_by_client
      t.string :created_by_user
      t.datetime :created_at
      t.string :modified_by_client
      t.string :modified_by_user
      t.datetime :modified_at
      t.string :name
      t.text :components

      t.timestamps
    end
    add_index :pipelines, :uuid, :unique => true
  end
  def down
    drop_table :pipelines
  end
end
