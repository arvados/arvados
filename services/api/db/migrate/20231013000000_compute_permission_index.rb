# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

class ComputePermissionIndex < ActiveRecord::Migration[5.2]
  def up
    # The inner part of compute_permission_subgraph has a query clause like this:
    #
    #    where u.perm_origin_uuid = m.target_uuid AND m.traverse_owned
    #         AND (m.user_uuid = m.target_uuid or m.target_uuid not like '_____-tpzed-_______________')
    #
    # This will end up doing a sequential scan on
    # materialized_permissions, which can easily have millions of
    # rows, unless we fully index the table for this query.  In one test,
    # this brought the compute_permission_subgraph query from over 6
    # seconds down to 250ms.
    #
    ActiveRecord::Base.connection.execute "drop index if exists index_materialized_permissions_target_is_not_user"
    ActiveRecord::Base.connection.execute %{
create index index_materialized_permissions_target_is_not_user on materialized_permissions (target_uuid, traverse_owned, (user_uuid = target_uuid or target_uuid not like '_____-tpzed-_______________'));
}
  end

  def down
    ActiveRecord::Base.connection.execute "drop index if exists index_materialized_permissions_target_is_not_user"
  end
end
