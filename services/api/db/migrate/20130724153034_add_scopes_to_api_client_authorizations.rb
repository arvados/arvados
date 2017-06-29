# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

class AddScopesToApiClientAuthorizations < ActiveRecord::Migration
  def change
    add_column :api_client_authorizations, :scopes, :text, :null => false, :default => ['all'].to_yaml
  end
end
