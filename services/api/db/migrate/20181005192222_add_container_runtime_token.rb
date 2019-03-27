# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

class AddContainerRuntimeToken < ActiveRecord::Migration[4.2]
  def change
    add_column :container_requests, :runtime_token, :text, :null => true
    add_column :containers, :runtime_user_uuid, :text, :null => true
    add_column :containers, :runtime_auth_scopes, :jsonb, :null => true
  end
end
