# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

class AddContainerLockCount < ActiveRecord::Migration
  def change
    add_column :containers, :lock_count, :int, :null => false, :default => 0
  end
end
