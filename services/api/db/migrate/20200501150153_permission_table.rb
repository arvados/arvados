class PermissionTable < ActiveRecord::Migration[5.0]
  def up
    create_table :materialized_permissions, :id => false do |t|
      t.string :user_uuid
      t.string :target_uuid
      t.integer :perm_level
      t.boolean :traverse_owned
    end
    add_index :materialized_permissions, [:user_uuid, :target_uuid], unique: true, name: 'permission_user_target'

    ActiveRecord::Base.connection.execute %{
create or replace function project_subtree (starting_uuid varchar(27))
returns table (target_uuid varchar(27))
STABLE
language SQL
as $$
WITH RECURSIVE
        project_subtree(uuid) as (
        values (starting_uuid)
        union
        select groups.uuid from groups join project_subtree on (groups.owner_uuid = project_subtree.uuid)
        )
        select uuid from project_subtree;
$$;
}

    ActiveRecord::Base.connection.execute %{
create or replace function project_subtree_with_trash_at (starting_uuid varchar(27), starting_trash_at timestamp)
returns table (target_uuid varchar(27), trash_at timestamp)
STABLE
language SQL
as $$
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

    create_table :trashed_groups, :id => false do |t|
      t.string :group_uuid
      t.datetime :trash_at
    end
    add_index :trashed_groups, :group_uuid, :unique => true

        ActiveRecord::Base.connection.execute %{
create or replace function compute_trashed ()
returns table (uuid varchar(27), trash_at timestamp)
STABLE
language SQL
as $$
select ps.target_uuid as group_uuid, ps.trash_at from groups,
  lateral project_subtree_with_trash_at(groups.uuid, groups.trash_at) ps
  where groups.owner_uuid like '_____-tpzed-_______________'
$$;
}

        ActiveRecord::Base.connection.execute("INSERT INTO trashed_groups select * from compute_trashed()")

        ActiveRecord::Base.connection.execute %{
create or replace function should_traverse_owned (starting_uuid varchar(27),
                                                  starting_perm integer)
  returns bool
STABLE
language SQL
as $$
select starting_uuid like '_____-j7d0g-_______________' or
       (starting_uuid like '_____-tpzed-_______________' and starting_perm >= 3);
$$;
}


        ActiveRecord::Base.connection.execute %{
create or replace function permission_graph_edges ()
  returns table (tail_uuid varchar(27), head_uuid varchar(27), val integer)
STABLE
language SQL
as $$
           select groups.owner_uuid, groups.uuid, (3) from groups
          union
            select users.owner_uuid, users.uuid, (3) from users
          union
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
$$;
}

        # Get a set of permission by searching the graph and following
        # ownership and permission links.
        #
        # edges() - a subselect with the union of ownership and permission links
        #
        # traverse_graph() - recursive query, from the starting node,
        # self-join with edges to find outgoing permissions.
        # Re-runs the query on new rows until there are no more results.
        # This accomplishes a breadth-first search of the permission graph.
        #
        ActiveRecord::Base.connection.execute %{
create or replace function search_permission_graph (starting_uuid varchar(27),
                                                    starting_perm integer)
  returns table (target_uuid varchar(27), val integer, traverse_owned bool)
STABLE
language SQL
as $$
WITH RECURSIVE edges(tail_uuid, head_uuid, val) as (
          select * from permission_graph_edges()
        ),
        traverse_graph(target_uuid, val, traverse_owned) as (
            values (starting_uuid, starting_perm,
                    should_traverse_owned(starting_uuid, starting_perm))
          union
            (select edges.head_uuid,
                    least(edges.val, traverse_graph.val,
                          case traverse_graph.traverse_owned
                            when true then null
                            else 0
                          end),
                    should_traverse_owned(edges.head_uuid, edges.val)
             from edges
             join traverse_graph on (traverse_graph.target_uuid = edges.tail_uuid)))
        select target_uuid, max(val), bool_or(traverse_owned) from traverse_graph
        group by (target_uuid) ;
$$;
}


  # owned_by_user_perms(perm_origin_uuid, target_uuid, val, traverse_owned) as (
  #   select users.owner_uuid as perm_origin_uuid, u.target_uuid, u.val, u.traverse_owned
  #     from users, lateral search_permission_graph(users.uuid, 3) as u
  #     where users.owner_uuid not in (select target_uuid from perm_from_start) and
  #           users.uuid in (select target_uuid from perm_from_start)
  # ),

  # owned_by_group_perms(perm_origin_uuid, target_uuid, val, traverse_owned) as (
  #   select groups.owner_uuid as perm_origin_uuid, groups.uuid, 3, true
  #     from groups
  #     where groups.owner_uuid not in (select target_uuid from perm_from_start) and
  #           groups.uuid in (select target_uuid from perm_from_start)
  # ),


        ActiveRecord::Base.connection.execute %{
create or replace function compute_permission_subgraph (perm_origin_uuid varchar(27),
                                                        starting_uuid varchar(27),
                                                        starting_perm integer)
returns table (user_uuid varchar(27), target_uuid varchar(27), val integer, traverse_owned bool)
STABLE
language SQL
as $$
with
perm_from_start(perm_origin_uuid, target_uuid, val, traverse_owned) as (
  select perm_origin_uuid, target_uuid, val, traverse_owned
    from search_permission_graph(starting_uuid, starting_perm)),

  edges(tail_uuid, head_uuid, val) as (
        select * from permission_graph_edges()),

  additional_perms(perm_origin_uuid, target_uuid, val, traverse_owned) as (
    select edges.tail_uuid as perm_origin_uuid, ps.target_uuid, ps.val,
           should_traverse_owned(ps.target_uuid, ps.val)
      from edges, lateral search_permission_graph(edges.head_uuid, edges.val) as ps
      where (not (edges.tail_uuid = perm_origin_uuid and
                 edges.head_uuid = starting_uuid and
                 edges.val = starting_perm)) and
            edges.tail_uuid not in (select target_uuid from perm_from_start) and
            edges.head_uuid in (select target_uuid from perm_from_start)),

  partial_perms(perm_origin_uuid, target_uuid, val, traverse_owned) as (
      select * from perm_from_start
    union
      select * from additional_perms
  ),

  user_identity_perms(perm_origin_uuid, target_uuid, val, traverse_owned) as (
    select users.uuid as perm_origin_uuid, ps.target_uuid, ps.val, ps.traverse_owned
      from users, lateral search_permission_graph(users.uuid, 3) as ps
      where users.owner_uuid not in (select target_uuid from partial_perms where traverse_owned) and
      users.uuid in (select target_uuid from partial_perms)
  ),

  all_perms(perm_origin_uuid, target_uuid, val, traverse_owned) as (
      select * from partial_perms
    union
      select * from user_identity_perms
  )

  select v.user_uuid, v.target_uuid, max(v.perm_level), bool_or(v.traverse_owned) from
    (select materialized_permissions.user_uuid,
         u.target_uuid,
         least(u.val, materialized_permissions.perm_level) as perm_level,
         u.traverse_owned
      from all_perms as u
      join materialized_permissions on (u.perm_origin_uuid = materialized_permissions.target_uuid)
      where materialized_permissions.traverse_owned
    union
      select perm_origin_uuid as user_uuid, target_uuid, val as perm_level, traverse_owned
        from all_perms
        where perm_origin_uuid like '_____-tpzed-_______________') as v
    group by v.user_uuid, v.target_uuid
$$;
     }

    ActiveRecord::Base.connection.execute "DROP MATERIALIZED VIEW IF EXISTS materialized_permission_view;"

  end
  def down
    drop_table :materialized_permissions
    drop_table :trashed_groups

    ActiveRecord::Base.connection.execute "DROP function project_subtree (varchar);"
    ActiveRecord::Base.connection.execute "DROP function project_subtree_with_trash_at (varchar, timestamp);"
    ActiveRecord::Base.connection.execute "DROP function compute_trashed ();"
    ActiveRecord::Base.connection.execute "DROP function search_permission_graph(varchar, integer);"
    ActiveRecord::Base.connection.execute "DROP function compute_permission_subgraph (varchar, varchar, integer);"
    ActiveRecord::Base.connection.execute "DROP function should_traverse_owned(varchar, integer);"
    ActiveRecord::Base.connection.execute "DROP function permission_graph_edges();"
  end
end
