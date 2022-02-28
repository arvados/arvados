# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

class AddFrozenByUuidToGroups < ActiveRecord::Migration[5.2]
  def change
    add_column :groups, :frozen_by_uuid, :string
  end
end
