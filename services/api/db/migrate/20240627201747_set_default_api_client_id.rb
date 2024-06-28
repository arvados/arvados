# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

class SetDefaultApiClientId < ActiveRecord::Migration[7.0]
  def change
    ActiveRecord::Base.connection.execute 'ALTER TABLE api_client_authorizations ALTER COLUMN api_client_id SET DEFAULT 0'
  end
end
