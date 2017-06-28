# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

class AddAuthsToContainer < ActiveRecord::Migration
  def change
    add_column :containers, :auth_uuid, :string
    add_column :containers, :locked_by_uuid, :string
  end
end
