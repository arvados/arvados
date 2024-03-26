# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

require_relative '20200501150153_permission_table_constants'

REVOKE_PERM = 0
CAN_MANAGE_PERM = 3

def update_permissions perm_origin_uuid, starting_uuid, perm_level, edge_id=nil, update_all_users=false
  return if Thread.current[:suppress_update_permissions]

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
  # check: for testing/debugging, compare the result of the
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

  if edge_id.nil?
    # For changes of ownership, edge_id is starting_uuid.  In turns
    # out most invocations of update_permissions are for changes of
    # ownership, so make this parameter optional to reduce
    # clutter.
    # For permission links, the uuid of the link object will be passed in for edge_id.
    edge_id = starting_uuid
  end

  ActiveRecord::Base.transaction do

    # "Conflicts with the ROW SHARE, ROW EXCLUSIVE, SHARE UPDATE
    # EXCLUSIVE, SHARE, SHARE ROW EXCLUSIVE, EXCLUSIVE, and ACCESS
    # EXCLUSIVE lock modes. This mode allows only concurrent ACCESS
    # SHARE locks, i.e., only reads from the table can proceed in
    # parallel with a transaction holding this lock mode."
    ActiveRecord::Base.connection.execute "LOCK TABLE #{PERMISSION_VIEW} in EXCLUSIVE MODE"

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
    # Update: as of 2023-10-13, incorrect merge join behavior is still
    # observed on at least one major user installation that is using
    # Postgres 14, so it seems this workaround is still needed.
    #
    # https://git.furworks.de/opensourcemirror/postgresql/commit/a314c34079cf06d05265623dd7c056f8fa9d577f
    #
    # Disable merge join for just this query (also local for this transaction), then reenable it.
    ActiveRecord::Base.connection.exec_query "SET LOCAL enable_mergejoin to false;"

    if perm_origin_uuid[5..11] == '-tpzed-' && !update_all_users
      # Modifying permission granted to a user, recompute the all permissions for that user

      ActiveRecord::Base.connection.exec_query %{
with origin_user_perms as (
    select pq.origin_uuid as user_uuid, target_uuid, pq.val, pq.traverse_owned from (
    #{PERM_QUERY_TEMPLATE % {:base_case => %{
        select '#{perm_origin_uuid}'::varchar(255), '#{perm_origin_uuid}'::varchar(255), 3, true, true
               where exists (select uuid from users where uuid='#{perm_origin_uuid}')
},
:edge_perm => %{
case (edges.edge_id = '#{edge_id}')
                               when true then #{perm_level}
                               else edges.val
                            end
}
} }) as pq),

/*
     Because users always have permission on themselves, this
     query also makes sure those permission rows are always
     returned.
*/
temptable_perms as (
      select * from origin_user_perms
    union all
      select target_uuid as user_uuid, target_uuid, 3, true
        from origin_user_perms
        where origin_user_perms.target_uuid like '_____-tpzed-_______________' and
              origin_user_perms.target_uuid != '#{perm_origin_uuid}'
),

/*
    Now that we have recomputed a set of permissions, delete any
    rows from the materialized_permissions table where (target_uuid,
    user_uuid) is not present or has perm_level=0 in the recomputed
    set.
*/
delete_rows as (
  delete from #{PERMISSION_VIEW} where
    user_uuid='#{perm_origin_uuid}' and
    not exists (select 1 from temptable_perms
                where target_uuid=#{PERMISSION_VIEW}.target_uuid and
                      user_uuid='#{perm_origin_uuid}' and
                      val>0)
)

/*
  Now insert-or-update permissions in the recomputed set.  The
  WHERE clause is important to avoid redundantly updating rows
  that haven't actually changed.
*/
insert into #{PERMISSION_VIEW} (user_uuid, target_uuid, perm_level, traverse_owned)
  select user_uuid, target_uuid, val as perm_level, traverse_owned from temptable_perms where val>0
on conflict (user_uuid, target_uuid) do update
set perm_level=EXCLUDED.perm_level, traverse_owned=EXCLUDED.traverse_owned
where #{PERMISSION_VIEW}.user_uuid=EXCLUDED.user_uuid and
      #{PERMISSION_VIEW}.target_uuid=EXCLUDED.target_uuid and
       (#{PERMISSION_VIEW}.perm_level != EXCLUDED.perm_level or
        #{PERMISSION_VIEW}.traverse_owned != EXCLUDED.traverse_owned);

}
    else
      # Modifying permission granted to a group, recompute permissions for everything accessible through that group
    ActiveRecord::Base.connection.exec_query %{
with temptable_perms as (
  select * from compute_permission_subgraph($1, $2, $3, $4)),

/*
    Now that we have recomputed a set of permissions, delete any
    rows from the materialized_permissions table where (target_uuid,
    user_uuid) is not present or has perm_level=0 in the recomputed
    set.
*/
delete_rows as (
  delete from #{PERMISSION_VIEW} where
    target_uuid in (select target_uuid from temptable_perms) and
    not exists (select 1 from temptable_perms
                where target_uuid=#{PERMISSION_VIEW}.target_uuid and
                      user_uuid=#{PERMISSION_VIEW}.user_uuid and
                      val>0)
)

/*
  Now insert-or-update permissions in the recomputed set.  The
  WHERE clause is important to avoid redundantly updating rows
  that haven't actually changed.
*/
insert into #{PERMISSION_VIEW} (user_uuid, target_uuid, perm_level, traverse_owned)
  select user_uuid, target_uuid, val as perm_level, traverse_owned from temptable_perms where val>0
on conflict (user_uuid, target_uuid) do update
set perm_level=EXCLUDED.perm_level, traverse_owned=EXCLUDED.traverse_owned
where #{PERMISSION_VIEW}.user_uuid=EXCLUDED.user_uuid and
      #{PERMISSION_VIEW}.target_uuid=EXCLUDED.target_uuid and
       (#{PERMISSION_VIEW}.perm_level != EXCLUDED.perm_level or
        #{PERMISSION_VIEW}.traverse_owned != EXCLUDED.traverse_owned);
},
                                             'update_permissions.select',
                                             [perm_origin_uuid,
                                              starting_uuid,
                                              perm_level,
                                              edge_id]
    end

    if perm_level>0
      check_permissions_against_full_refresh
    end
  end
end


def check_permissions_against_full_refresh
  # No-op except when running tests
  return unless Rails.env == 'test' and !Thread.current[:no_check_permissions_against_full_refresh] and !Thread.current[:suppress_update_permissions]

  # For checking correctness of the incremental permission updates.
  # Check contents of the current 'materialized_permission' table
  # against a from-scratch permission refresh.

  q1 = ActiveRecord::Base.connection.exec_query %{
select user_uuid, target_uuid, perm_level, traverse_owned from #{PERMISSION_VIEW}
order by user_uuid, target_uuid
}, "check_permissions_against_full_refresh.permission_table"

  q2 = ActiveRecord::Base.connection.exec_query %{
    select pq.origin_uuid as user_uuid, target_uuid, pq.val as perm_level, pq.traverse_owned from (
    #{PERM_QUERY_TEMPLATE % {:base_case => %{
        select uuid, uuid, 3, true, true from users
},
:edge_perm => 'edges.val'
} }) as pq order by origin_uuid, target_uuid
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

def skip_check_permissions_against_full_refresh
  check_perm_was = Thread.current[:no_check_permissions_against_full_refresh]
  Thread.current[:no_check_permissions_against_full_refresh] = true
  begin
    yield
  ensure
    Thread.current[:no_check_permissions_against_full_refresh] = check_perm_was
  end
end

def batch_update_permissions
  check_perm_was = Thread.current[:suppress_update_permissions]
  Thread.current[:suppress_update_permissions] = true
  begin
    yield
  ensure
    Thread.current[:suppress_update_permissions] = check_perm_was
    refresh_permissions
  end
end

# Used to account for permissions that a user gains by having
# can_manage on another user.
#
# note: in theory a user could have can_manage access to a user
# through multiple levels, that isn't handled here (would require a
# recursive query).  I think that's okay because users getting
# transitive access through "can_manage" on a user is is rarely/never
# used feature and something we probably want to deprecate and remove.
USER_UUIDS_SUBQUERY_TEMPLATE = %{
select target_uuid from materialized_permissions where user_uuid in (%{user})
and target_uuid like '_____-tpzed-_______________' and traverse_owned=true and perm_level >= %{perm_level}
}
