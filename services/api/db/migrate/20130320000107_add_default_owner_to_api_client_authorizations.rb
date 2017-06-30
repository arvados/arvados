# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

class AddDefaultOwnerToApiClientAuthorizations < ActiveRecord::Migration
  def change
    add_column :api_client_authorizations, :default_owner, :string
  end
end
