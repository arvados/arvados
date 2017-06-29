# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

class CreateApiClientAuthorizations < ActiveRecord::Migration
  def change
    create_table :api_client_authorizations do |t|
      t.string :api_token, :null => false
      t.references :api_client, :null => false
      t.references :user, :null => false
      t.string :created_by_ip_address
      t.string :last_used_by_ip_address
      t.datetime :last_used_at
      t.datetime :expires_at

      t.timestamps
    end
    add_index :api_client_authorizations, :api_token, :unique => true
    add_index :api_client_authorizations, :api_client_id
    add_index :api_client_authorizations, :user_id
    add_index :api_client_authorizations, :expires_at
  end
end
