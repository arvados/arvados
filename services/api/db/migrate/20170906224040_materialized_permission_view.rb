class MaterializedPermissionView < ActiveRecord::Migration

  @@idxtables = [:collections, :container_requests, :groups, :jobs, :links, :pipeline_instances, :pipeline_templates, :repositories, :users, :virtual_machines, :workflows]

  def up
    ActiveRecord::Base.connection.execute(
    "CREATE MATERIALIZED VIEW permission_view AS
WITH RECURSIVE
perm_value (name, val) AS (
     VALUES
     ('can_read',   1::smallint),
     ('can_login',  1),
     ('can_write',  2),
     ('can_manage', 3)
     ),
perm_edges (tail_uuid, head_uuid, val, follow, trashed) AS (
       SELECT links.tail_uuid,
              links.head_uuid,
              pv.val,
              (pv.val = 3 OR groups.uuid IS NOT NULL) AS follow,
              0::smallint AS trashed
              FROM links
              LEFT JOIN perm_value pv ON pv.name = links.name
              LEFT JOIN groups ON pv.val<3 AND groups.uuid = links.head_uuid
              WHERE links.link_class = 'permission'
       UNION ALL
       SELECT owner_uuid, uuid, 3, true,
              CASE WHEN trash_at IS NOT NULL and trash_at < clock_timestamp() THEN 1 ELSE 0 END
              FROM groups
       ),
perm (val, follow, user_uuid, target_uuid, trashed, startnode) AS (
     SELECT 3::smallint             AS val,
            false                   AS follow,
            users.uuid::varchar(32) AS user_uuid,
            users.uuid::varchar(32) AS target_uuid,
            0::smallint             AS trashed,
            true                    AS startnode
            FROM users
     UNION
     SELECT LEAST(perm.val, edges.val)::smallint  AS val,
            edges.follow                          AS follow,
            perm.user_uuid::varchar(32)           AS user_uuid,
            edges.head_uuid::varchar(32)          AS target_uuid,
            GREATEST(perm.trashed, edges.trashed)::smallint AS trashed,
            false                                 AS startnode
            FROM perm
            INNER JOIN perm_edges edges
            ON (perm.startnode or perm.follow) AND edges.tail_uuid = perm.target_uuid
)
SELECT user_uuid,
       target_uuid,
       MAX(val) AS perm_level,
       CASE follow WHEN true THEN target_uuid ELSE NULL END AS target_owner_uuid,
       MAX(trashed) AS trashed
       FROM perm
       GROUP BY user_uuid, target_uuid, target_owner_uuid;
")
    add_index :permission_view, [:trashed, :target_uuid], name: 'permission_target_trashed'
    add_index :permission_view, [:user_uuid, :trashed, :perm_level], name: 'permission_target_user_trashed_level'

    @@idxtables.each do |table|
      ActiveRecord::Base.connection.execute("CREATE INDEX index_#{table.to_s}_on_modified_at_uuid ON #{table.to_s} USING btree (modified_at desc, uuid asc)")
    end
  end

  def down
    remove_index :permission_view, name: 'permission_target_trashed'
    remove_index :permission_view, name: 'permission_target_user_trashed_level'
    @@idxtables.each do |table|
      ActiveRecord::Base.connection.execute("DROP INDEX index_#{table.to_s}_on_modified_at_uuid")
    end
    ActiveRecord::Base.connection.execute("DROP MATERIALIZED VIEW IF EXISTS permission_view")
  end
end
