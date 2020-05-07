class PermissionTable < ActiveRecord::Migration[5.0]
  def up
    create_table :materialized_permissions, :id => false do |t|
      t.string :user_uuid
      t.string :target_uuid
      t.integer :perm_level
      t.boolean :traverse_owned
    end

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
create or replace function search_permission_graph (starting_uuid varchar(27), starting_perm integer)
returns table (target_uuid varchar(27), val integer, traverse_owned bool)
STABLE
language SQL
as $$
WITH RECURSIVE edges(tail_uuid, head_uuid, val) as (
            select groups.owner_uuid, groups.uuid, (3) from groups
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
        ),
        traverse_graph(target_uuid, val, traverse_owned) as (
            values (starting_uuid, starting_perm, true)
          union
            (select edges.head_uuid,
                    least(edges.val, traverse_graph.val),
                    (edges.head_uuid like '_____-j7d0g-_______________' or
                     (edges.head_uuid like '_____-tpzed-_______________' and edges.val >= 3))
             from edges
             join traverse_graph on (traverse_graph.target_uuid = edges.tail_uuid)
             where traverse_graph.traverse_owned))
        select target_uuid, max(val), bool_or(traverse_owned) from traverse_graph
        group by (target_uuid) ;
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
  end
end
