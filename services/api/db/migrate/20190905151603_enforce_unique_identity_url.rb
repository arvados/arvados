# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

class EnforceUniqueIdentityUrl < ActiveRecord::Migration[5.0]
  def change
    add_index :users, [:identity_url], :unique => true
  end
end
