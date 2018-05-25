# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

class ChangeContainerPriorityBigint < ActiveRecord::Migration
  def change
    change_column :containers, :priority, :integer, limit: 8
  end
end
