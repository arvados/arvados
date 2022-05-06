# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

class AddOutputProperties < ActiveRecord::Migration[5.2]
  def change
    add_column :container_requests, :output_properties, :jsonb, default: {}
    add_column :containers, :output_properties, :jsonb, default: {}
  end
end
