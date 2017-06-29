# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

class RemoveNameFromCollections < ActiveRecord::Migration
  def up
    remove_column :collections, :name
  end

  def down
    add_column :collections, :name, :string
  end
end
