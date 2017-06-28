# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

class AddIsTrustedToApiClients < ActiveRecord::Migration
  def change
    add_column :api_clients, :is_trusted, :boolean, :default => false
  end
end
