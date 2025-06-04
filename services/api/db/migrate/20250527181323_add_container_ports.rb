# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

class AddContainerPorts < ActiveRecord::Migration[7.1]
  def change
    create_table :container_ports, :id => false do |t|
      t.integer :external_port, :null => false
      t.integer :container_port, :null => false
      t.string :container_uuid, :null => false
    end
    add_index :container_ports, :external_port, unique: true
    add_index :container_ports, :container_uuid
  end
end
