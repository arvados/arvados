# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

PERMISSION_VIEW = "materialized_permissions"
TRASHED_GROUPS = "trashed_groups"

def refresh_permission_view
  ActiveRecord::Base.transaction do
    ActiveRecord::Base.connection.execute("LOCK TABLE #{PERMISSION_VIEW}")
    ActiveRecord::Base.connection.execute("DELETE FROM #{PERMISSION_VIEW}")
    ActiveRecord::Base.connection.execute %{
INSERT INTO #{PERMISSION_VIEW}
select users.uuid, g.target_uuid, g.val, g.traverse_owned
from users, lateral search_permission_graph(users.uuid, 3) as g where g.val > 0
},
                                          "refresh_permission_view.do"
  end
end

def refresh_trashed
  ActiveRecord::Base.transaction do
    ActiveRecord::Base.connection.execute("LOCK TABLE #{TRASHED_GROUPS}")
    ActiveRecord::Base.connection.execute("DELETE FROM #{TRASHED_GROUPS}")
    ActiveRecord::Base.connection.execute("INSERT INTO #{TRASHED_GROUPS} select * from compute_trashed()")
  end
end

def update_permissions perm_origin_uuid, starting_uuid, perm_level, check=false
  #
  # Update a subset of the permission table affected by adding or
  # removing a particular permission relationship (ownership or a
  # permission link).
  #
  # perm_origin_uuid: This is the object that 'gets' the permission.
  # It is the owner_uuid or tail_uuid.
  #
  # starting_uuid: The object we are computing permission for (or head_uuid)
  #
  # perm_level: The level of permission that perm_origin_uuid gets for starting_uuid.
  #
  # perm_level is a number from 0-3
  #   can_read=1
  #   can_write=2
  #   can_manage=3
  #   or call with perm_level=0 to revoke permissions
  #
  # check: for testing/debugging only, compare the result of the
  # incremental update against a full table recompute.  Throws an
  # error if the contents are not identical (ie they produce different
  # permission results)

  # Theory of operation
  #
  # Give a change in a specific permission relationship, we recompute
  # the set of permissions (for all users) that could possibly be
  # affected by that relationship.  For example, if a project is
  # shared with another user, we recompute all permissions for all
  # projects in the hierarchy.  This returns a set of updated
  # permissions, which we stash in a temporary table.
  #
  # Then, for each user_uuid/target_uuid in the updated permissions
  # result set we insert/update a permission row in
  # materialized_permissions, and delete any rows that exist in
  # materialized_permissions that are not in the result set or have
  # perm_level=0.
  #
  # see db/migrate/20200501150153_permission_table.rb for details on
  # how the permissions are computed.

  # "Conflicts with the ROW EXCLUSIVE, SHARE UPDATE EXCLUSIVE, SHARE
  # ROW EXCLUSIVE, EXCLUSIVE, and ACCESS EXCLUSIVE lock modes. This
  # mode protects a table against concurrent data changes."
  ActiveRecord::Base.connection.execute "LOCK TABLE #{PERMISSION_VIEW} in SHARE MODE"

  # Workaround for
  # BUG #15160: planner overestimates number of rows in join when there are more than 200 rows coming from CTE
  # https://www.postgresql.org/message-id/152395805004.19366.3107109716821067806@wrigleys.postgresql.org
  #
  # For a crucial join in the compute_permission_subgraph() query, the
  # planner mis-estimates the number of rows in a Common Table
  # Expression (CTE, this is a subquery in a WITH clause) and as a
  # result it chooses the wrong join order.  The join starts with the
  # permissions table because it mistakenly thinks
  # count(materalized_permissions) < count(new computed permissions)
  # when actually it is the other way around.
  #
  # Because of the incorrect join order, it choose the wrong join
  # strategy (merge join, which works best when two tables are roughly
  # the same size).  As a workaround, we can tell it not to use that
  # join strategy, this causes it to pick hash join instead, which
  # turns out to be a bit better.  However, because the join order is
  # still wrong, we don't get the full benefit of the index.
  #
  # This is very unfortunate because it makes the query performance
  # dependent on the size of the materalized_permissions table, when
  # the goal of this design was to make permission updates scale-free
  # and only depend on the number of permissions affected and not the
  # total table size.  In several hours of researching I wasn't able
  # to find a way to force the correct join order, so I'm calling it
  # here and I have to move on.
  #
  # This is apparently addressed in Postgres 12, but I developed &
  # tested this on Postgres 9.6, so in the future we should reevaluate
  # the performance & query plan on Postgres 12.
  #
  # https://git.furworks.de/opensourcemirror/postgresql/commit/a314c34079cf06d05265623dd7c056f8fa9d577f
  #
  # Disable merge join for just this query (also local for this transaction), then reenable it.
  ActiveRecord::Base.connection.exec_query "SET LOCAL enable_mergejoin to false;"

  temptable_perms = "temp_perms_#{rand(2**64).to_s(10)}"
  ActiveRecord::Base.connection.exec_query %{
create temporary table #{temptable_perms} on commit drop
as select * from compute_permission_subgraph($1, $2, $3)
},
                                           'update_permissions.select',
                                           [[nil, perm_origin_uuid],
                                            [nil, starting_uuid],
                                            [nil, perm_level]]

  ActiveRecord::Base.connection.exec_query "SET LOCAL enable_mergejoin to true;"

  ActiveRecord::Base.connection.exec_delete %{
delete from #{PERMISSION_VIEW} where
  target_uuid in (select target_uuid from #{temptable_perms}) and
  not exists (select 1 from #{temptable_perms}
              where target_uuid=#{PERMISSION_VIEW}.target_uuid and
                    user_uuid=#{PERMISSION_VIEW}.user_uuid and
                    val>0)
},
                                        "update_permissions.delete"

  ActiveRecord::Base.connection.exec_query %{
insert into #{PERMISSION_VIEW} (user_uuid, target_uuid, perm_level, traverse_owned)
  select user_uuid, target_uuid, val as perm_level, traverse_owned from #{temptable_perms} where val>0
on conflict (user_uuid, target_uuid) do update set perm_level=EXCLUDED.perm_level, traverse_owned=EXCLUDED.traverse_owned;
},
                                           "update_permissions.insert"

  if check and perm_level>0
    check_permissions_against_full_refresh
  end
end


def check_permissions_against_full_refresh
  #
  # For debugging, this checks contents of the
  # incrementally-updated 'materialized_permission' against a
  # from-scratch permission refresh.
  #

  q1 = ActiveRecord::Base.connection.exec_query %{
select user_uuid, target_uuid, perm_level, traverse_owned from #{PERMISSION_VIEW}
order by user_uuid, target_uuid
}, "check_permissions_against_full_refresh.permission_table"

  q2 = ActiveRecord::Base.connection.exec_query %{
select users.uuid as user_uuid, g.target_uuid, g.val as perm_level, g.traverse_owned
from users, lateral search_permission_graph(users.uuid, 3) as g where g.val > 0
order by users.uuid, target_uuid
}, "check_permissions_against_full_refresh.full_recompute"

  if q1.count != q2.count
    puts "Didn't match incremental+: #{q1.count} != full refresh-: #{q2.count}"
  end

  if q1.count > q2.count
    q1.each_with_index do |r, i|
      if r != q2[i]
        puts "+#{r}\n-#{q2[i]}"
        raise "Didn't match"
      end
    end
  else
    q2.each_with_index do |r, i|
      if r != q1[i]
        puts "+#{q1[i]}\n-#{r}"
        raise "Didn't match"
      end
    end
  end
end
