# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

class AddPortableDataHashIndexToCollections < ActiveRecord::Migration
  def change
    add_index :collections, :portable_data_hash
  end
end
