# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

class AddRefreshesAtToApiClientAuthorizations < ActiveRecord::Migration[7.1]
  def change
    add_column :api_client_authorizations, :refreshes_at, :timestamp, null: true
    add_index :api_client_authorizations, :refreshes_at
  end
end
