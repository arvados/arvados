# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

class SerializedColumnsAcceptNull < ActiveRecord::Migration
  def change
    change_column :api_client_authorizations, :scopes, :text, null: true, default: '["all"]'
  end
end
