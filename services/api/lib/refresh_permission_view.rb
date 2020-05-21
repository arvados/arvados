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
  ActiveRecord::Base.connection.execute("DELETE FROM #{TRASHED_GROUPS}")
  ActiveRecord::Base.connection.execute("INSERT INTO #{TRASHED_GROUPS} select * from compute_trashed()")
end

def update_permissions perm_origin_uuid, starting_uuid, perm_level, check=false
  # Update a subset of the permission graph
  # perm_level is the inherited permission
  # perm_level is a number from 0-3
  #   can_read=1
  #   can_write=2
  #   can_manage=3
  #   call with perm_level=0 to revoke permissions
  #
  # 1. Compute set (group, permission) implied by traversing
  #    graph starting at this group
  # 2. Find links from outside the graph that point inside
  # 3. For each starting uuid, get the set of permissions from the
  #    materialized permission table
  # 3. Delete permissions from table not in our computed subset.
  # 4. Upsert each permission in our subset (user, group, val)

  ActiveRecord::Base.connection.execute "LOCK TABLE #{PERMISSION_VIEW} in SHARE MODE"

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
