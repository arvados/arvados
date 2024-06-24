# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

class CreateUuidLocks < ActiveRecord::Migration[7.0]
  def change
    create_table :uuid_locks, id: false do |t|
      t.string :uuid, null: false, index: {unique: true}
      t.integer :n, null: false, default: 0
    end
  end
end
