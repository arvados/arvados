# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

require '20200501150153_permission_table_constants'

class PermissionTable < ActiveRecord::Migration[5.0]
  def up
    # This is a major migration.  We are replacing the
    # materialized_permission_view, which is fully recomputed any time
    # a permission changes (and becomes very expensive as the number
    # of users/groups becomes large), with a new strategy that only
    # recomputes permissions for the subset of objects that are
    # potentially affected by the addition or removal of a permission
    # relationship (i.e. ownership or a permission link).
    #
    # This also disentangles the concept of "trashed groups" from the
    # permissions system.  Updating trashed items follows a similar
    # (but less complicated) strategy to updating permissions, so it
    # may be helpful to look at that first.
    #

    ActiveRecord::Base.connection.execute "DROP MATERIALIZED VIEW IF EXISTS materialized_permission_view;"
    drop_table :permission_refresh_lock

    # This table stores the set of trashed groups and their trash_at
    # time.  Used to exclude trashed projects and their contents when
    # getting object listings.
    create_table :trashed_groups, :id => false do |t|
      t.string :group_uuid
      t.datetime :trash_at
    end
    add_index :trashed_groups, :group_uuid, :unique => true

    ActiveRecord::Base.connection.execute %{
create or replace function project_subtree_with_trash_at (starting_uuid varchar(27), starting_trash_at timestamp)
returns table (target_uuid varchar(27), trash_at timestamp)
STABLE
language SQL
as $$
/* Starting from a project, recursively traverse all the projects
  underneath it and return a set of project uuids and trash_at times
  (may be null).  The initial trash_at can be a timestamp or null.
  The trash_at time propagates downward to groups it owns, i.e. when a
  group is trashed, everything underneath it in the ownership
  hierarchy is also considered trashed.  However, this is fact is
  recorded in the trashed_groups table, not by updating trash_at field
  in the groups table.
*/
WITH RECURSIVE
        project_subtree(uuid, trash_at) as (
        values (starting_uuid, starting_trash_at)
        union
        select groups.uuid, LEAST(project_subtree.trash_at, groups.trash_at)
          from groups join project_subtree on (groups.owner_uuid = project_subtree.uuid)
        )
        select uuid, trash_at from project_subtree;
$$;
}

    # Now populate the table.  For a non-test databse this is the only
    # time this ever happens, after this the trash table is updated
    # incrementally.  See app/models/group.rb#update_trash
    refresh_trashed

    # The table to store the flattened permissions.  This is almost
    # exactly the same as the old materalized_permission_view except
    # that the target_owner_uuid colunm in the view is now just a
    # boolean traverse_owned (the column was only ever tested for null
    # or non-null).
    #
    # For details on how this table is used to apply permissions to
    # queries, see app/models/arvados_model.rb#readable_by
    #
    create_table :materialized_permissions, :id => false do |t|
      t.string :user_uuid
      t.string :target_uuid
      t.integer :perm_level
      t.boolean :traverse_owned
    end
    add_index :materialized_permissions, [:user_uuid, :target_uuid], unique: true, name: 'permission_user_target'
    add_index :materialized_permissions, [:target_uuid], unique: false, name: 'permission_target'

    ActiveRecord::Base.connection.execute %{
create or replace function should_traverse_owned (starting_uuid varchar(27),
                                                  starting_perm integer)
  returns bool
IMMUTABLE
language SQL
as $$
/* Helper function.  Determines if permission on an object implies
   transitive permission to things the object owns.  This is always
   true for groups, but only true for users when the permission level
   is can_manage.
*/
select starting_uuid like '_____-j7d0g-_______________' or
       (starting_uuid like '_____-tpzed-_______________' and starting_perm >= 3);
$$;
}

    # Merge all permission relationships into a single view.  This
    # consists of: groups (projects) owning things, users owning
    # things, users owning themselves, and explicit permission links.
    #
    # A SQL view gets inlined into the query where it is used as a
    # subquery.  This enables the query planner to inject constraints,
    # so we only look up edges we plan to traverse and avoid a brute
    # force query of all edges.
    ActiveRecord::Base.connection.execute %{
create view permission_graph_edges as
  select groups.owner_uuid as tail_uuid, groups.uuid as head_uuid, (3) as val from groups
union all
  select users.owner_uuid as tail_uuid, users.uuid as head_uuid, (3) as val from users
union all
  select users.uuid as tail_uuid, users.uuid as head_uuid, (3) as val from users
union all
  select links.tail_uuid,
         links.head_uuid,
         CASE
           WHEN links.name = 'can_read'   THEN 1
           WHEN links.name = 'can_login'  THEN 1
           WHEN links.name = 'can_write'  THEN 2
           WHEN links.name = 'can_manage' THEN 3
           ELSE 0
          END as val
      from links
      where links.link_class='permission'
}

    # Code fragment that is used below.  This is used to ensure that
    # the permission edge passed into compute_permission_subgraph
    # takes precedence over an existing edge in the "edges" view.
    override = %{,
                            case (edges.tail_uuid = perm_origin_uuid AND
                                  edges.head_uuid = starting_uuid)
                               when true then starting_perm
                               else null
                            end
}

    #
    # The primary function to compute permissions for a subgraph.
    # This originally was organized somewhat more cleanly, but this
    # ran into performance issues due to the query optimizer not
    # working across function and "with" expression boundaries.  So I
    # had to fall back on using string templates for the repeated
    # code.  I'm sorry.

    ActiveRecord::Base.connection.execute %{
create or replace function compute_permission_subgraph (perm_origin_uuid varchar(27),
                                                        starting_uuid varchar(27),
                                                        starting_perm integer)
returns table (user_uuid varchar(27), target_uuid varchar(27), val integer, traverse_owned bool)
STABLE
language SQL
as $$

/* The purpose of this function is to compute the permissions for a
   subgraph of the database, starting from a given edge.  The newly
   computed permissions are used to add and remove rows from the main
   permissions table.

   perm_origin_uuid: The object that 'gets' the permission.

   starting_uuid: The starting object the permission applies to.

   starting_perm: The permission that perm_origin_uuid 'has' on
                  starting_uuid One of 1, 2, 3 for can_read,
                  can_write, can_manage respectively, or 0 to revoke
                  permissions.
*/
with
  /* Starting from starting_uuid, determine the set of objects that
     could be affected by this permission change.

     Note: We don't traverse users unless it is an "identity"
     permission (permission origin is self).
  */
  perm_from_start(perm_origin_uuid, target_uuid, val, traverse_owned) as (
    #{PERM_QUERY_TEMPLATE % {:base_case => %{
             values (perm_origin_uuid, starting_uuid, starting_perm,
                    should_traverse_owned(starting_uuid, starting_perm),
                    (perm_origin_uuid = starting_uuid or starting_uuid not like '_____-tpzed-_______________'))
},
:override => override
} }),

  /* Find other inbound edges that grant permissions to 'targets' in
     perm_from_start, and compute permissions that originate from
     those.

     This is necessary for two reasons:

       1) Other users may have access to a subset of the objects
       through other permission links than the one we started from.
       If we don't recompute them, their permission will get dropped.

       2) There may be more than one path through which a user gets
       permission to an object.  For example, a user owns a project
       and also shares it can_read with a group the user belongs
       to. adding the can_read link must not overwrite the existing
       can_manage permission granted by ownership.
  */
  additional_perms(perm_origin_uuid, target_uuid, val, traverse_owned) as (
    #{PERM_QUERY_TEMPLATE % {:base_case => %{
    select edges.tail_uuid as origin_uuid, edges.head_uuid as target_uuid, edges.val,
           should_traverse_owned(edges.head_uuid, edges.val),
           edges.head_uuid like '_____-j7d0g-_______________'
      from permission_graph_edges as edges
      where (not (edges.tail_uuid = perm_origin_uuid and
                  edges.head_uuid = starting_uuid)) and
            edges.tail_uuid not in (select target_uuid from perm_from_start where target_uuid like '_____-j7d0g-_______________') and
            edges.head_uuid in (select target_uuid from perm_from_start)
},
:override => override
} }),

  /* Combine the permissions computed in the first two phases. */
  all_perms(perm_origin_uuid, target_uuid, val, traverse_owned) as (
      select * from perm_from_start
    union all
      select * from additional_perms
  )

  /* The actual query that produces rows to be added or removed
     from the materialized_permissions table.  This is the clever
     bit.

     Key insights:

     * For every group, the materialized_permissions lists all users
       that can access to that group.

     * The all_perms subquery has computed permissions on on a set of
       objects for all inbound "origins", which are users or groups.

     * Permissions through groups are transitive.

     We can infer:

     1) The materialized_permissions table declares that user X has permission N on group Y
     2) The all_perms result has determined group Y has permission M on object Z
     3) Therefore, user X has permission min(N, M) on object Z

     This allows us to efficiently determine the set of users that
     have permissions on the subset of objects, without having to
     follow the chain of permission back up to find those users.

     In addition, because users always have permission on themselves, this
     query also makes sure those permission rows are always
     returned.
  */
  select v.user_uuid, v.target_uuid, max(v.perm_level), bool_or(v.traverse_owned) from
    (select m.user_uuid,
         u.target_uuid,
         least(u.val, m.perm_level) as perm_level,
         u.traverse_owned
      from all_perms as u, materialized_permissions as m
           where u.perm_origin_uuid = m.target_uuid AND m.traverse_owned
           AND (m.user_uuid = m.target_uuid or m.target_uuid not like '_____-tpzed-_______________')
    union all
      select target_uuid as user_uuid, target_uuid, 3, true
        from all_perms
        where all_perms.target_uuid like '_____-tpzed-_______________') as v
    group by v.user_uuid, v.target_uuid
$$;
     }

    #
    # Populate materialized_permissions by traversing permissions
    # starting at each user.
    #
    refresh_permissions
  end

  def down
    drop_table :materialized_permissions
    drop_table :trashed_groups

    ActiveRecord::Base.connection.execute "DROP function project_subtree_with_trash_at (varchar, timestamp);"
    ActiveRecord::Base.connection.execute "DROP function compute_permission_subgraph (varchar, varchar, integer);"
    ActiveRecord::Base.connection.execute "DROP function should_traverse_owned(varchar, integer);"
    ActiveRecord::Base.connection.execute "DROP view permission_graph_edges;"

    ActiveRecord::Base.connection.execute(%{
CREATE MATERIALIZED VIEW materialized_permission_view AS
 WITH RECURSIVE perm_value(name, val) AS (
         VALUES ('can_read'::text,(1)::smallint), ('can_login'::text,1), ('can_write'::text,2), ('can_manage'::text,3)
        ), perm_edges(tail_uuid, head_uuid, val, follow, trashed) AS (
         SELECT links.tail_uuid,
            links.head_uuid,
            pv.val,
            ((pv.val = 3) OR (groups.uuid IS NOT NULL)) AS follow,
            (0)::smallint AS trashed,
            (0)::smallint AS followtrash
           FROM ((public.links
             LEFT JOIN perm_value pv ON ((pv.name = (links.name)::text)))
             LEFT JOIN public.groups ON (((pv.val < 3) AND ((groups.uuid)::text = (links.head_uuid)::text))))
          WHERE ((links.link_class)::text = 'permission'::text)
        UNION ALL
         SELECT groups.owner_uuid,
            groups.uuid,
            3,
            true AS bool,
                CASE
                    WHEN ((groups.trash_at IS NOT NULL) AND (groups.trash_at < clock_timestamp())) THEN 1
                    ELSE 0
                END AS "case",
            1
           FROM public.groups
        ), perm(val, follow, user_uuid, target_uuid, trashed) AS (
         SELECT (3)::smallint AS val,
            true AS follow,
            (users.uuid)::character varying(32) AS user_uuid,
            (users.uuid)::character varying(32) AS target_uuid,
            (0)::smallint AS trashed
           FROM public.users
        UNION
         SELECT (LEAST((perm_1.val)::integer, edges.val))::smallint AS val,
            edges.follow,
            perm_1.user_uuid,
            (edges.head_uuid)::character varying(32) AS target_uuid,
            ((GREATEST((perm_1.trashed)::integer, edges.trashed) * edges.followtrash))::smallint AS trashed
           FROM (perm perm_1
             JOIN perm_edges edges ON ((perm_1.follow AND ((edges.tail_uuid)::text = (perm_1.target_uuid)::text))))
        )
 SELECT perm.user_uuid,
    perm.target_uuid,
    max(perm.val) AS perm_level,
        CASE perm.follow
            WHEN true THEN perm.target_uuid
            ELSE NULL::character varying
        END AS target_owner_uuid,
    max(perm.trashed) AS trashed
   FROM perm
  GROUP BY perm.user_uuid, perm.target_uuid,
        CASE perm.follow
            WHEN true THEN perm.target_uuid
            ELSE NULL::character varying
        END
  WITH NO DATA;
}
    )

    add_index :materialized_permission_view, [:trashed, :target_uuid], name: 'permission_target_trashed'
    add_index :materialized_permission_view, [:user_uuid, :trashed, :perm_level], name: 'permission_target_user_trashed_level'
    create_table :permission_refresh_lock

    ActiveRecord::Base.connection.execute 'REFRESH MATERIALIZED VIEW materialized_permission_view;'
  end
end
