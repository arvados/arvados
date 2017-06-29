# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

class AddGroupClassToGroups < ActiveRecord::Migration
  def change
    add_column :groups, :group_class, :string
    add_index :groups, :group_class
  end
end
