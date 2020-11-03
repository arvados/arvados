# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

require '20200501150153_permission_table_constants'

class RefreshTrashedGroups < ActiveRecord::Migration[5.2]
  def change
    # The original refresh_trashed query had a bug, it would insert
    # all trashed rows, including those with null trash_at times.
    # This went unnoticed because null trash_at behaved the same as
    # not having those rows at all, but it is inefficient to fetch
    # rows we'll never use.  That bug is fixed in the original query
    # but we need another migration to make sure it runs.
    refresh_trashed
  end
end
