# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

class AddSecretMountsToContainers < ActiveRecord::Migration
  def change
    add_column :container_requests, :secret_mounts, :jsonb, default: {}
    add_column :containers, :secret_mounts, :jsonb, default: {}
    add_column :containers, :secret_mounts_md5, :string, default: "99914b932bd37a50b983c5e7c90ae93b"
    add_index :containers, :secret_mounts_md5
  end
end
