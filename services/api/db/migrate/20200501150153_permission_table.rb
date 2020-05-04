class PermissionTable < ActiveRecord::Migration[5.0]
  def up
    create_table :materialized_permissions, :id => false do |t|
      t.string :user_uuid
      t.string :target_uuid
      t.integer :perm_level
      t.string :target_owner_uuid
    end

    ActiveRecord::Base.connection.execute %{
create or replace function compute_permission_table ()
returns table(user_uuid character varying (27),
              target_uuid character varying (27),
              perm_level smallint,
              target_owner_uuid character varying(27))
VOLATILE
language SQL
as $$
 WITH RECURSIVE perm_value(name, val) AS (
         VALUES ('can_read'::text,(1)::smallint), ('can_login'::text,1), ('can_write'::text,2), ('can_manage'::text,3)
        ), perm_edges(tail_uuid, head_uuid, val, follow) AS (
         SELECT links.tail_uuid,
            links.head_uuid,
            pv.val,
            ((pv.val = 3) OR (groups.uuid IS NOT NULL)) AS follow
           FROM ((public.links
             LEFT JOIN perm_value pv ON ((pv.name = (links.name)::text)))
             LEFT JOIN public.groups ON (((pv.val < 3) AND ((groups.uuid)::text = (links.head_uuid)::text))))
          WHERE ((links.link_class)::text = 'permission'::text)
        UNION ALL
         SELECT groups.owner_uuid,
            groups.uuid,
            3,
            true AS bool
           FROM public.groups
        ), perm(val, follow, user_uuid, target_uuid) AS (
         SELECT (3)::smallint AS val,
            true AS follow,
            (users.uuid)::character varying(32) AS user_uuid,
            (users.uuid)::character varying(32) AS target_uuid
           FROM public.users
        UNION
         SELECT (LEAST((perm_1.val)::integer, edges.val))::smallint AS val,
            edges.follow,
            perm_1.user_uuid,
            (edges.head_uuid)::character varying(32) AS target_uuid
           FROM (perm perm_1
             JOIN perm_edges edges ON ((perm_1.follow AND ((edges.tail_uuid)::text = (perm_1.target_uuid)::text))))
        )
 SELECT perm.user_uuid,
    perm.target_uuid,
    max(perm.val) AS perm_level,
    CASE perm.follow
       WHEN true THEN perm.target_uuid
       ELSE NULL::character varying
    END AS target_owner_uuid
   FROM perm
  GROUP BY perm.user_uuid, perm.target_uuid,
        CASE perm.follow
            WHEN true THEN perm.target_uuid
            ELSE NULL::character varying
        END
$$;
}

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
create or replace function project_subtree_notrash (starting_uuid varchar(27))
returns table (target_uuid varchar(27))
STABLE
language SQL
as $$
WITH RECURSIVE
	project_subtree(uuid) as (
	values (starting_uuid)
	union
	select groups.uuid from groups join project_subtree on (groups.owner_uuid = project_subtree.uuid)
        where groups.is_trashed=false
	)
	select uuid from project_subtree;
$$;
}

    create_table :trashed_groups, :id => false do |t|
      t.string :uuid
    end

        ActiveRecord::Base.connection.execute %{
create or replace function compute_trashed ()
returns table (uuid varchar(27))
STABLE
language SQL
as $$
select ps.target_uuid from groups,
  lateral project_subtree(groups.uuid) ps
  where groups.is_trashed = true
$$;
}

    ActiveRecord::Base.connection.execute "DROP MATERIALIZED VIEW IF EXISTS materialized_permission_view;"

  end
  def down
    drop_table :materialized_permissions
    drop_table :trashed_groups
  end
end
