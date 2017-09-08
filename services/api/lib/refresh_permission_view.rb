# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

PERMISSION_VIEW = "materialized_permission_view"

def refresh_permission_view
  ActiveRecord::Base.connection.exec_query("REFRESH MATERIALIZED VIEW #{PERMISSION_VIEW}")
end
