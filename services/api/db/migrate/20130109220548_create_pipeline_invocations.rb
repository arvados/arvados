# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

class CreatePipelineInvocations < ActiveRecord::Migration
  def up
    create_table :pipeline_invocations do |t|
      t.string :uuid
      t.string :created_by_client
      t.string :created_by_user
      t.datetime :created_at
      t.string :modified_by_client
      t.string :modified_by_user
      t.datetime :modified_at
      t.string :pipeline_uuid
      t.string :name
      t.text :components
      t.boolean :success, :null => true
      t.boolean :active, :default => false

      t.timestamps
    end
    add_index :pipeline_invocations, :uuid, :unique => true
  end
  def down
    drop_table :pipeline_invocations
  end
end
