# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

class RenameUserDefaultOwner < ActiveRecord::Migration
  def change
    rename_column :users, :default_owner, :default_owner_uuid
  end
end
