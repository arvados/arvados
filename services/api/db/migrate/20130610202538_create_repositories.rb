# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

class CreateRepositories < ActiveRecord::Migration
  def change
    create_table :repositories do |t|
      t.string :uuid, :null => false
      t.string :owner, :null => false
      t.string :modified_by_client
      t.string :modified_by_user
      t.datetime :modified_at
      t.string :name
      t.string :fetch_url
      t.string :push_url

      t.timestamps
    end
    add_index :repositories, :uuid, :unique => true
    add_index :repositories, :name
  end
end
