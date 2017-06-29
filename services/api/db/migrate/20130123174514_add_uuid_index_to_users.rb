# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

class AddUuidIndexToUsers < ActiveRecord::Migration
  def change
    add_index :users, :uuid, :unique => true
  end
end
