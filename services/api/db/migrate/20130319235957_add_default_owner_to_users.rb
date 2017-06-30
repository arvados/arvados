# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

class AddDefaultOwnerToUsers < ActiveRecord::Migration
  def change
    add_column :users, :default_owner, :string
  end
end
