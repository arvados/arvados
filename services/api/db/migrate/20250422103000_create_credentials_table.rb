# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

class CreateCredentialsTable < ActiveRecord::Migration[7.1]
  def change
    create_table :credentials, :id => :string, :primary_key => :uuid do |t|
      t.string :owner_uuid, :null => false
      t.datetime :created_at, :null => false
      t.datetime :modified_at, :null => false
      t.string :modified_by_client_uuid
      t.string :modified_by_user_uuid
      t.string :name
      t.text :description
      t.string :credential_class
      t.string :credential_id
      t.text :credential_secret
      t.datetime :expires_at, :null => false
    end
    add_index :credentials, :uuid, unique: true
    add_index :credentials, :owner_uuid
    add_index :credentials, [:owner_uuid, :name], unique: true
    add_index :credentials, [:uuid, :owner_uuid, :modified_by_client_uuid, :modified_by_user_uuid, :name, :credential_class, :credential_id]
  end
end
