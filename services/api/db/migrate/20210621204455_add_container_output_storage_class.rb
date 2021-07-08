# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

class AddContainerOutputStorageClass < ActiveRecord::Migration[5.2]
  def change
    add_column :container_requests, :output_storage_classes, :jsonb, :default => ["default"]
    add_column :containers, :output_storage_classes, :jsonb, :default => ["default"]
  end
end
