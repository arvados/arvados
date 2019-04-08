# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

class AddVersionInfoToCollections < ActiveRecord::Migration[4.2]
  def change
    # Do changes in bulk to save time on huge tables
    change_table :collections, :bulk => true do |t|
      t.string :current_version_uuid
      t.integer :version, null: false, default: 1
      t.index [:current_version_uuid, :version], unique: true
    end
  end
end
