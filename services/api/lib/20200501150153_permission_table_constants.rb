# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

# These constants are used in both
# db/migrate/20200501150153_permission_table and update_permissions
#
# This file allows them to be easily imported by both to avoid duplication.
#
# Don't mess with this!  Any changes will affect both the current
# update_permissions and the past migration.  If you are tinkering
# with the permission system and need to change how
# PERM_QUERY_TEMPLATE, refresh_trashed or refresh_permissions works,
# you should make a new file with your modified functions and have
# update_permissions reference that file instead.

PERMISSION_VIEW = "materialized_permissions"
TRASHED_GROUPS = "trashed_groups"
FROZEN_GROUPS = "frozen_groups"

# We need to use this parameterized query in a few different places,
# including as a subquery in a larger query.
#
# There's basically two options, the way I did this originally was to
# put this in a postgres function and do a lateral join over it.
# However, postgres functions impose an optimization barrier, and
# possibly have other overhead with temporary tables, so I ended up
# going with the brute force approach of inlining the whole thing.
#
# The two substitutions are "base_case" which determines the initial
# set of permission origins and "edge_perm" which is used to ensure
# that the new permission takes precedence over the one in the edges
# table (but some queries don't need that.)
#
PERM_QUERY_TEMPLATE = %{
WITH RECURSIVE
        traverse_graph(origin_uuid, target_uuid, val, traverse_owned, starting_set) as (
            %{base_case}
          union
            (select traverse_graph.origin_uuid,
                    edges.head_uuid,
                      least(%{edge_perm},
                            traverse_graph.val),
                    should_traverse_owned(edges.head_uuid, edges.val),
                    false
             from permission_graph_edges as edges, traverse_graph
             where traverse_graph.target_uuid = edges.tail_uuid
             and (edges.tail_uuid like '_____-j7d0g-_______________' or
                  traverse_graph.starting_set)))
        select traverse_graph.origin_uuid, target_uuid, max(val) as val, bool_or(traverse_owned) as traverse_owned from traverse_graph
        group by (traverse_graph.origin_uuid, target_uuid)
}

def refresh_trashed
  ActiveRecord::Base.transaction do
    ActiveRecord::Base.connection.execute("LOCK TABLE #{TRASHED_GROUPS}")
    ActiveRecord::Base.connection.execute("DELETE FROM #{TRASHED_GROUPS}")

    # Helper populate trashed_groups table. This starts with
    #   each group owned by a user and computes the subtree under that
    #   group to find any groups that are trashed.
    ActiveRecord::Base.connection.execute(%{
INSERT INTO #{TRASHED_GROUPS}
select ps.target_uuid as group_uuid, ps.trash_at from groups,
  lateral project_subtree_with_trash_at(groups.uuid, groups.trash_at) ps
  where groups.owner_uuid like '_____-tpzed-_______________' and ps.trash_at is not NULL
})
  end
end

def refresh_permissions
  ActiveRecord::Base.transaction do
    ActiveRecord::Base.connection.execute("LOCK TABLE #{PERMISSION_VIEW}")
    ActiveRecord::Base.connection.execute("DELETE FROM #{PERMISSION_VIEW}")

    ActiveRecord::Base.connection.execute %{
INSERT INTO materialized_permissions
    #{PERM_QUERY_TEMPLATE % {:base_case => %{
        select uuid, uuid, 3, true, true from users
},
:edge_perm => 'edges.val'
} }
}, "refresh_permission_view.do"
  end
end

def refresh_frozen
  ActiveRecord::Base.transaction do
    ActiveRecord::Base.connection.execute("LOCK TABLE #{FROZEN_GROUPS}")
    ActiveRecord::Base.connection.execute("DELETE FROM #{FROZEN_GROUPS}")

    # Compute entire frozen_groups table, starting with top-level
    # projects (i.e., project groups owned by a user).
    ActiveRecord::Base.connection.execute(%{
INSERT INTO #{FROZEN_GROUPS}
select ps.uuid from groups,
  lateral project_subtree_with_is_frozen(groups.uuid, groups.frozen_by_uuid is not null) ps
  where groups.owner_uuid like '_____-tpzed-_______________'
    and group_class = 'project'
    and ps.is_frozen
})
  end
end
