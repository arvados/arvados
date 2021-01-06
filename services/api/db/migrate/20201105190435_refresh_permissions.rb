# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

require '20200501150153_permission_table_constants'

class RefreshPermissions < ActiveRecord::Migration[5.2]
  def change
    # There was a report of deadlocks resulting in failing permission
    # updates.  These failures should not have corrupted permissions
    # (the failure should have rolled back the entire update) but we
    # will refresh the permissions out of an abundance of caution.
    refresh_permissions
  end
end
