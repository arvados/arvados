# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

class ReadOnlyOnKeepServices < ActiveRecord::Migration
  def change
    add_column :keep_services, :read_only, :boolean, null: false, default: false
  end
end
