-- Copyright (C) The Arvados Authors. All rights reserved.
--
-- SPDX-License-Identifier: AGPL-3.0

CREATE TEMPORARY VIEW ancestor_view AS
WITH RECURSIVE
ancestor (uuid, ancestor_uuid) AS (
     SELECT groups.uuid::varchar(32)       AS uuid,
            groups.owner_uuid::varchar(32) AS ancestor_uuid
            FROM groups
     UNION
     SELECT ancestor.uuid::varchar(32)     AS uuid,
            groups.owner_uuid::varchar(32) AS ancestor_uuid
            FROM ancestor
            INNER JOIN groups
            ON groups.uuid = ancestor.ancestor_uuid
)
SELECT * FROM ancestor;
