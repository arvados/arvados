# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

class CreateApiClients < ActiveRecord::Migration
  def change
    create_table :api_clients do |t|
      t.string :uuid
      t.string :created_by_client
      t.string :created_by_user
      t.string :modified_by_client
      t.string :modified_by_user
      t.datetime :modified_at
      t.string :name
      t.string :url_prefix

      t.timestamps
    end
    add_index :api_clients, :uuid, :unique => true
  end
end
