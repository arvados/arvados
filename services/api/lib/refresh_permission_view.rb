# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

PERMISSION_VIEW = "materialized_permissions"
TRASHED_GROUPS = "trashed_groups"

def do_refresh_permission_view
  ActiveRecord::Base.transaction do
    ActiveRecord::Base.connection.execute("LOCK TABLE permission_refresh_lock")
    ActiveRecord::Base.connection.execute("DELETE FROM #{PERMISSION_VIEW}")
    ActiveRecord::Base.connection.execute %{
INSERT INTO #{PERMISSION_VIEW}
select users.uuid, g.target_uuid, g.val, g.traverse_owned
from users, lateral search_permission_graph(users.uuid, 3) as g
}
  end
end

def refresh_permission_view(async=false)
  if async and Rails.configuration.API.AsyncPermissionsUpdateInterval > 0
    exp = Rails.configuration.API.AsyncPermissionsUpdateInterval.seconds
    need = false
    Rails.cache.fetch('AsyncRefreshPermissionView', expires_in: exp) do
      need = true
    end
    if need
      # Schedule a new permission update and return immediately
      Thread.new do
        Thread.current.abort_on_exception = false
        begin
          sleep(exp)
          Rails.cache.delete('AsyncRefreshPermissionView')
          do_refresh_permission_view
        rescue => e
          Rails.logger.error "Updating permission view: #{e}\n#{e.backtrace.join("\n\t")}"
        ensure
          ActiveRecord::Base.connection.close
        end
      end
      true
    end
  else
    do_refresh_permission_view
  end
end
