# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

class AddStorageClassesToCollections < ActiveRecord::Migration
  def up
    add_column :collections, :storage_classes_desired, :jsonb, :default => ["default"]
    add_column :collections, :storage_classes_confirmed, :jsonb, :default => []
    add_column :collections, :storage_classes_confirmed_at, :datetime, :default => nil, :null => true
  end

  def down
    remove_column :collections, :storage_classes_desired
    remove_column :collections, :storage_classes_confirmed
    remove_column :collections, :storage_classes_confirmed_at
  end
end
