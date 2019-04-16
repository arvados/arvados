# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

class AddIsActiveToUsers < ActiveRecord::Migration[4.2]
  def change
    add_column :users, :is_active, :boolean, :default => false
  end
end
