# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

class AddFileInfoToCollection < ActiveRecord::Migration
  def change
    add_column :collections, :file_count, :integer, default: 0, null: false
    add_column :collections, :file_size_total, :integer, default: 0, null: false
  end
end
