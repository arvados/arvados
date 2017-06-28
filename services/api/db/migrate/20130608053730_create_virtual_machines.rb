# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

class CreateVirtualMachines < ActiveRecord::Migration
  def change
    create_table :virtual_machines do |t|
      t.string :uuid, :null => false
      t.string :owner, :null => false
      t.string :modified_by_client
      t.string :modified_by_user
      t.datetime :modified_at
      t.string :hostname

      t.timestamps
    end
    add_index :virtual_machines, :uuid, :unique => true
    add_index :virtual_machines, :hostname
  end
end
