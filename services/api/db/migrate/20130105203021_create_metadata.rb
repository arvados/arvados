# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

class CreateMetadata < ActiveRecord::Migration
  def change
    create_table :metadata do |t|
      t.string :uuid
      t.string :created_by_client
      t.string :created_by_user
      t.datetime :created_at
      t.string :modified_by_client
      t.string :modified_by_user
      t.datetime :modified_at
      t.string :target_uuid
      t.string :target_kind
      t.references :native_target, :polymorphic => true
      t.string :metadatum_class
      t.string :key
      t.string :value
      t.text :info # "unlimited length" in postgresql

      t.timestamps
    end
  end
end
