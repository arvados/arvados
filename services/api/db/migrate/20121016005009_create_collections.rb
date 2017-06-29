# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

class CreateCollections < ActiveRecord::Migration
  def change
    create_table :collections do |t|
      t.string :locator
      t.string :create_by_client
      t.string :created_by_user
      t.datetime :created_at
      t.string :modified_by_client
      t.string :modified_by_user
      t.datetime :modified_at
      t.string :portable_data_hash
      t.string :name
      t.integer :redundancy
      t.string :redundancy_confirmed_by_client
      t.datetime :redundancy_confirmed_at
      t.integer :redundancy_confirmed_as

      t.timestamps
    end
  end
end
