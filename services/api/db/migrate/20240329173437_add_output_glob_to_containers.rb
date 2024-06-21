# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

class AddOutputGlobToContainers < ActiveRecord::Migration[7.0]
  def change
    add_column :containers, :output_glob, :text, default: '[]'
    add_column :container_requests, :output_glob, :text, default: '[]'
  end
end
