# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

PERMISSION_VIEW = "materialized_permission_view"

def do_refresh_permission_view
  ActiveRecord::Base.transaction do
    ActiveRecord::Base.connection.execute("LOCK TABLE permission_refresh_lock")
    ActiveRecord::Base.connection.execute("REFRESH MATERIALIZED VIEW #{PERMISSION_VIEW}")
  end
end

def refresh_permission_view(async=false)
  if async and Rails.configuration.async_permissions_update_interval > 0
    exp = Rails.configuration.async_permissions_update_interval.seconds
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
