# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

class AddCostToContainers < ActiveRecord::Migration[5.2]
  def change
    add_column :containers, :cost, :float, null: false, default: 0
    add_column :containers, :subrequests_cost, :float, null: false, default: 0
    add_column :container_requests, :cumulative_cost, :float, null: false, default: 0
  end
end
