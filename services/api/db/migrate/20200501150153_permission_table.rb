# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

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

    ActiveRecord::Base.connection.execute %{
create or replace function compute_trashed ()
returns table (uuid varchar(27), trash_at timestamp)
STABLE
language SQL
as $$
/* Helper function to populate trashed_groups table. This starts with
   each group owned by a user and computes the subtree under that
   group to find any groups that are trashed.
*/
select ps.target_uuid as group_uuid, ps.trash_at from groups,
  lateral project_subtree_with_trash_at(groups.uuid, groups.trash_at) ps
  where groups.owner_uuid like '_____-tpzed-_______________'
$$;
}

    # Now populate the table.  For a non-test databse this is the only
    # time this ever happens, after this the trash table is updated
    # incrementally.  See app/models/group.rb#update_trash
    ActiveRecord::Base.connection.execute("INSERT INTO trashed_groups select * from compute_trashed()")

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
    # things, and explicit permission links.
    #
    # Fun fact, a SQL view gets inlined into the query where it is
    # used, this enables the query planner to inject constraints, so
    # when using the view we only look up edges we plan to traverse
    # and avoid a brute force computation of all edges.
    ActiveRecord::Base.connection.execute %{
create view permission_graph_edges as
  select groups.owner_uuid as tail_uuid, groups.uuid as head_uuid, (3) as val from groups
union all
  select users.owner_uuid as tail_uuid, users.uuid as head_uuid, (3) as val from users
union all
  select links.tail_uuid,
         links.head_uuid,
         CASE
           WHEN links.name = 'can_read'   THEN 1
           WHEN links.name = 'can_login'  THEN 1
           WHEN links.name = 'can_write'  THEN 2
           WHEN links.name = 'can_manage' THEN 3
          END as val
      from links
      where links.link_class='permission'
}

    ActiveRecord::Base.connection.execute %{
create or replace function search_permission_graph (starting_uuid varchar(27),
                                                    starting_perm integer,
                                                    override_edge_tail varchar(27) default null,
                                                    override_edge_head varchar(27) default null,
                                                    override_edge_perm integer default null)
  returns table (target_uuid varchar(27), val integer, traverse_owned bool)
STABLE
language SQL
as $$
/*
  From starting_uuid, perform a recursive self-join on the edges
  to follow chains of permissions.  This is a breadth-first search
  of the permission graph.  Permission is propagated across edges,
  which may narrow the permission for subsequent links (eg I start
  at can_manage but when traversing a can_read link everything
  touched through that link will only be can_read).

  When revoking a permission, we follow the chain of permissions but
  with a permissions level of 0.  The update on the permissions table
  has to happen _before_ the permission is actually removed, because
  we need to be able to traverse the edge before it goes away.  When
  we do that, we also need to traverse it at the _new_ permission
  level - this is what override_edge_tail/head/perm are for.

  Yields the set of objects that are potentially affected, and
  their permission levels granted by having starting_perm on
  starting_uuid.

  If starting_uuid is a user, this computes the entire set of
  permissions for that user (because it returns everything that is
  reachable by that user).

  Used by the compute_permission_subgraph function.
*/
WITH RECURSIVE
        traverse_graph(target_uuid, val, traverse_owned) as (
            values (starting_uuid, starting_perm,
                    should_traverse_owned(starting_uuid, starting_perm))
          union
            (select edges.head_uuid,
                      least(edges.val,
                            traverse_graph.val,
                            case traverse_graph.traverse_owned
                              when true then null
                              else 0
                            end,
                            case (edges.tail_uuid = override_edge_tail AND
                                  edges.head_uuid = override_edge_head)
                               when true then override_edge_perm
                               else null
                            end),
                    should_traverse_owned(edges.head_uuid, edges.val)
             from permission_graph_edges as edges, traverse_graph
             where traverse_graph.target_uuid = edges.tail_uuid))
        select target_uuid, max(val), bool_or(traverse_owned) from traverse_graph
        group by (target_uuid);
$$;
}

    ActiveRecord::Base.connection.execute %{
create or replace function compute_permission_subgraph (perm_origin_uuid varchar(27),
                                                        starting_uuid varchar(27),
                                                        starting_perm integer)
returns table (user_uuid varchar(27), target_uuid varchar(27), val integer, traverse_owned bool)
STABLE
language SQL
as $$
/* perm_origin_uuid: The object that 'gets' or 'has' the permission.

   starting_uuid: The starting object the permission applies to.

   starting_perm: The permission that perm_origin_uuid 'has' on
                  starting_uuid One of 1, 2, 3 for can_read,
                  can_write, can_manage respectively, or 0 to revoke
                  permissions.

   This function is broken up into a number of clauses, described
   below.

   Note on query optimization:

   Each clause in a "with" statement is called a "common table
   expression" or CTE.

   In Postgres, they are evaluated in sequence and results of each CTE
   is stored in a temporary table.  This means Postgres does not
   propagate constraints from later subqueries to earlier subqueries
   when they are CTEs.

   This is a problem if, for example, a later subquery chooses 10
   items out of a set of 1000000 defined by an earlier subquery,
   because it will always compute all 1000000 rows even if the query
   on the 1000000 rows could have been constrained.  This is why
   permission_graph_edges is a view -- views are inlined so and can be
   optimized using external constraints.

   The query optimizer does sort the temporary tables for later use in
   joins.

   Final note, this query would have been almost impossible to write
   (and certainly impossible to read) without splitting it up using
   SQL "with" but unfortunately it also stumbles into a frustrating
   Postgres optimizer bug, see
   lib/refresh_permission_view.rb#update_permissions
   for details and a partial workaround.
*/
with
  /* Gets the initial set of objects potentially affected by the
     permission change, using search_permission_graph.
  */
  perm_from_start(perm_origin_uuid, target_uuid, val, traverse_owned) as (
    select perm_origin_uuid, target_uuid, val, traverse_owned
      from search_permission_graph(starting_uuid,
                                   starting_perm,
                                   perm_origin_uuid,
                                   starting_uuid,
                                   starting_perm)),

  /* Finds other inbound edges that grant permissions on the objects
     in perm_from_start, and computes permissions that originate from
     those.  This is required to handle the case where there is more
     than one path through which a user gets permission to an object.
     For example, a user owns a project and also shares it can_read
     with a group the user belongs to, adding the can_read link must
     not overwrite the existing can_manage permission granted by
     ownership.
  */
  additional_perms(perm_origin_uuid, target_uuid, val, traverse_owned) as (
    select edges.tail_uuid as perm_origin_uuid, ps.target_uuid, ps.val,
           should_traverse_owned(ps.target_uuid, ps.val)
      from permission_graph_edges as edges,
           lateral search_permission_graph(edges.head_uuid,
                                           edges.val,
                                           perm_origin_uuid,
                                           starting_uuid,
                                           starting_perm) as ps
      where (not (edges.tail_uuid = perm_origin_uuid and
                 edges.head_uuid = starting_uuid)) and
            edges.tail_uuid not in (select target_uuid from perm_from_start) and
            edges.head_uuid in (select target_uuid from perm_from_start)),

  /* Combines the permissions computed in the first two phases. */
  partial_perms(perm_origin_uuid, target_uuid, val, traverse_owned) as (
      select * from perm_from_start
    union all
      select * from additional_perms
  ),

  /* If there are any users in the set of potentially affected objects
     and the user's owner was not traversed, recompute permissions for
     that user.  This is required because users always have permission
     to themselves (identity property) which would be missing from the
     permission set if the user was traversed while computing
     permissions for another object.
  */
  user_identity_perms(perm_origin_uuid, target_uuid, val, traverse_owned) as (
    select users.uuid as perm_origin_uuid, ps.target_uuid, ps.val, ps.traverse_owned
      from users, lateral search_permission_graph(users.uuid,
                                                  3,
                                                  perm_origin_uuid,
                                                  starting_uuid,
                                                  starting_perm) as ps
      where (users.owner_uuid not in (select target_uuid from partial_perms) or
             users.owner_uuid = users.uuid) and
      users.uuid in (select target_uuid from partial_perms)
  ),

  /* Combines all the computed permissions into one table. */
  all_perms(perm_origin_uuid, target_uuid, val, traverse_owned) as (
      select * from partial_perms
    union
      select * from user_identity_perms
  )

  /* The actual query that produces rows to be added or removed
     from the materialized_permissions table.  This is the clever
     bit.

     Key insights:

     * Permissions are transitive (with some special cases involving
       users, this is controlled by the traverse_owned flag).

     * A user object can only gain permissions via an inbound edge,
       or appearing in the graph.

     * The materialized_permissions table includes the permission
       each user has on the tail end of each inbound edge.

     * The all_perms subquery has permissions for each object in the
       subgraph reachable from certain origin (tail end of an edge).

     * Therefore, for each user, we can compute user permissions on
       each object in subgraph by determining the permission the user
       has on each origin (tail end of an edge), joining that with the
       perm_origin_uuid column of all_perms, and taking the least() of
       the origin edge or all_perms val (because of the "least
       permission on the path" rule).  If an object was reachable by
       more than one path (appears with more than one origin), we take
       the max() of the computed permissions.

     * Finally, because users always have permission on themselves, the
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
    union all
      select perm_origin_uuid as user_uuid, target_uuid, val as perm_level, traverse_owned
        from all_perms
        where all_perms.perm_origin_uuid like '_____-tpzed-_______________') as v
    group by v.user_uuid, v.target_uuid
$$;
     }

    #
    # Populate the materialized_permissions by traversing permissions
    # starting at each user.
    #
    ActiveRecord::Base.connection.execute %{
INSERT INTO materialized_permissions
select users.uuid, g.target_uuid, g.val, g.traverse_owned
from users, lateral search_permission_graph(users.uuid, 3) as g where g.val > 0
}
  end

  def down
    drop_table :materialized_permissions
    drop_table :trashed_groups

    ActiveRecord::Base.connection.execute "DROP function project_subtree_with_trash_at (varchar, timestamp);"
    ActiveRecord::Base.connection.execute "DROP function compute_trashed ();"
    ActiveRecord::Base.connection.execute "DROP function search_permission_graph(varchar, integer, varchar, varchar, integer);"
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
