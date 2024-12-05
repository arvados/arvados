# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

class IndexOnContainerRequestName < ActiveRecord::Migration[7.0]
  def up
    old_value = query_value('SHOW statement_timeout')
    execute "SET statement_timeout TO '0'"

    add_index :container_requests, ["name", "owner_uuid"]

    execute "SET statement_timeout TO #{quote(old_value)}"
  end

  def down
    remove_index :container_requests, ["name", "owner_uuid"]
  end
end
