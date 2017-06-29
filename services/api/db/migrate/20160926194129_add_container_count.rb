# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

class AddContainerCount < ActiveRecord::Migration
  def up
    add_column :container_requests, :container_count, :int, :default => 0
  end

  def down
    remove_column :container_requests, :container_count
  end
end
