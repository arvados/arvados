# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

class AddOutputTtlToContainerRequests < ActiveRecord::Migration
  def change
    add_column :container_requests, :output_ttl, :integer, default: 0, null: false
  end
end
