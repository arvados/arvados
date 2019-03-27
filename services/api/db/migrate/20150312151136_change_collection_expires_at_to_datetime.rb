# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

class ChangeCollectionExpiresAtToDatetime < ActiveRecord::Migration[4.2]
  def up
    change_column :collections, :expires_at, :datetime
  end

  def down
    change_column :collections, :expires_at, :date
  end
end
