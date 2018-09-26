# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

class AddRuntimeStatusToContainers < ActiveRecord::Migration
  def change
    add_column :containers, :runtime_status, :jsonb, default: {}
    add_index :containers, :runtime_status, using: :gin
  end
end
