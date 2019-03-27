# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

class UpdateNodesIndex < ActiveRecord::Migration[4.2]
  def up
    remove_index :nodes, :hostname
    add_index :nodes, :hostname
  end
  def down
    remove_index :nodes, :hostname
    add_index :nodes, :hostname, :unique => true
  end
end
