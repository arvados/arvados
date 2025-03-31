# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

class AddServicesAndPublishedPorts < ActiveRecord::Migration[7.1]
  def change
    add_column :containers, :service, :boolean, null: false, :default => false
    add_column :container_requests, :service, :boolean, null: false, :default => false

    add_column :containers, :published_ports, :jsonb, :default => {}
    add_column :container_requests, :published_ports, :jsonb, :default => {}

    add_index :links, :name, :where => "link_class = 'published_port'", :unique => true
  end
end
