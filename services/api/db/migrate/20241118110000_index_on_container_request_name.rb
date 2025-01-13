# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

class IndexOnContainerRequestName < ActiveRecord::Migration[7.0]
  def up
    add_index :container_requests, ["name", "owner_uuid"]
  end

  def down
    remove_index :container_requests, ["name", "owner_uuid"]
  end
end
