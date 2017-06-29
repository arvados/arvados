# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

class AllUsersCanReadAnonymousGroup < ActiveRecord::Migration
  include CurrentApiClient

  def up
    anonymous_group_read_permission
  end

  def down
    # Do nothing - it's too dangerous to try to figure out whether or not
    # the permission was created by the migration.
  end
end
