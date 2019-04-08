# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

class AddContainerAuthUuidIndex < ActiveRecord::Migration[4.2]
  def change
    add_index :containers, :auth_uuid
  end
end
