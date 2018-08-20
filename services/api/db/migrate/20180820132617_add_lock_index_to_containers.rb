# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

class AddLockIndexToContainers < ActiveRecord::Migration
  def change
    # For the current code in sdk/go/dispatch:
    add_index :containers, [:locked_by_uuid, :priority]
    # For future dispatchers that use filters instead of offset for
    # more predictable paging:
    add_index :containers, [:locked_by_uuid, :uuid]
  end
end
