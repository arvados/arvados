# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

class AddOutputIsPersistentToJob < ActiveRecord::Migration
  def change
    add_column :jobs, :output_is_persistent, :boolean, null: false, default: false
  end
end
