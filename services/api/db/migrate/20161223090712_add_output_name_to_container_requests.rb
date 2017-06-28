# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

class AddOutputNameToContainerRequests < ActiveRecord::Migration
  def up
    add_column :container_requests, :output_name, :string, :default => nil
  end

  def down
    remove_column :container_requests, :output_name
  end
end
