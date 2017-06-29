# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

class AddPropertiesToNode < ActiveRecord::Migration
  def up
    add_column :nodes, :properties, :text
  end

  def down
    remove_column :nodes, :properties
  end
end
