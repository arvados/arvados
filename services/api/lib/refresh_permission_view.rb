# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

PERMISSION_VIEW = "materialized_permission_view"

def refresh_permission_view
  ActiveRecord::Base.transaction do
    ActiveRecord::Base.connection.execute("LOCK TABLE permission_refresh_lock")
    ActiveRecord::Base.connection.execute("REFRESH MATERIALIZED VIEW #{PERMISSION_VIEW}")
  end
end
