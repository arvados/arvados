# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

class AddObjectOwnerIndexToLogs < ActiveRecord::Migration
  def change
    add_index :logs, :object_owner_uuid
  end
end
