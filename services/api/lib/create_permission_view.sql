-- Copyright (C) The Arvados Authors. All rights reserved.
--
-- SPDX-License-Identifier: AGPL-3.0

-- Note: this is not the current code used for permission checks (that is
-- materialized_permission_view), but is retained here for migration purposes.

CREATE TEMPORARY VIEW permission_view AS
WITH RECURSIVE
perm_value (name, val) AS (
     VALUES
     ('can_read',   1::smallint),
     ('can_login',  1),
     ('can_write',  2),
     ('can_manage', 3)
     ),
perm_edges (tail_uuid, head_uuid, val, follow) AS (
       SELECT links.tail_uuid,
              links.head_uuid,
              pv.val,
              (pv.val = 3 OR groups.uuid IS NOT NULL) AS follow
              FROM links
              LEFT JOIN perm_value pv ON pv.name = links.name
              LEFT JOIN groups ON pv.val<3 AND groups.uuid = links.head_uuid
              WHERE links.link_class = 'permission'
       UNION ALL
       SELECT owner_uuid, uuid, 3, true FROM groups
       ),
perm (val, follow, user_uuid, target_uuid) AS (
     SELECT 3::smallint             AS val,
            true                    AS follow,
            users.uuid::varchar(32) AS user_uuid,
            users.uuid::varchar(32) AS target_uuid
            FROM users
     UNION
     SELECT LEAST(perm.val, edges.val)::smallint AS val,
            edges.follow                         AS follow,
            perm.user_uuid::varchar(32)          AS user_uuid,
            edges.head_uuid::varchar(32)         AS target_uuid
            FROM perm
            INNER JOIN perm_edges edges
            ON perm.follow AND edges.tail_uuid = perm.target_uuid
)
SELECT user_uuid,
       target_uuid,
       val AS perm_level,
       CASE follow WHEN true THEN target_uuid ELSE NULL END AS target_owner_uuid
       FROM perm;
