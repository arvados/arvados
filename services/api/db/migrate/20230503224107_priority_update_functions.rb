# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

class PriorityUpdateFunctions < ActiveRecord::Migration[5.2]
  def up
    ActiveRecord::Base.connection.execute %{
CREATE OR REPLACE FUNCTION container_priority(for_container_uuid character varying, inherited bigint, inherited_from character varying) returns bigint
    LANGUAGE sql
    AS $$
/* Determine the priority of an individual container.
   The "inherited" priority comes from the path we followed from the root, the parent container
   priority hasn't been updated in the table yet but we need to behave it like it has been.
*/
select coalesce(max(case when container_requests.priority = 0 then 0
                         when containers.uuid = inherited_from then inherited
                         when containers.priority is not NULL then containers.priority
                         else container_requests.priority * 1125899906842624::bigint - (extract(epoch from container_requests.created_at)*1000)::bigint
                    end), 0) from
    container_requests left outer join containers on container_requests.requesting_container_uuid = containers.uuid
    where container_requests.container_uuid = for_container_uuid and container_requests.state = 'Committed' and container_requests.priority > 0;
$$;
}

    ActiveRecord::Base.connection.execute %{
CREATE OR REPLACE FUNCTION update_priorities(for_container_uuid character varying) returns table (pri_container_uuid character varying, upd_priority bigint)
    LANGUAGE sql
    AS $$
/* Calculate the priorities of all containers starting from for_container_uuid.
   This traverses the process tree downward and calls container_priority for each container
   and returns a table of container uuids and their new priorities.
*/
with recursive tab(upd_container_uuid, upd_priority) as (
  select for_container_uuid, container_priority(for_container_uuid, 0, '')
union
  select containers.uuid, container_priority(containers.uuid, child_requests.upd_priority, child_requests.upd_container_uuid)
  from (tab join container_requests on tab.upd_container_uuid = container_requests.requesting_container_uuid) as child_requests
  join containers on child_requests.container_uuid = containers.uuid
  where containers.state in ('Queued', 'Locked', 'Running')
)
select upd_container_uuid, upd_priority from tab;
$$;
}

    ActiveRecord::Base.connection.execute %{
CREATE OR REPLACE FUNCTION container_tree(for_container_uuid character varying) returns table (pri_container_uuid character varying)
    LANGUAGE sql
    AS $$
/* A lighter weight version of the update_priorities query that only returns the containers in a tree,
   used by SELECT FOR UPDATE.
*/
with recursive tab(upd_container_uuid) as (
  select for_container_uuid
union
  select containers.uuid
  from (tab join container_requests on tab.upd_container_uuid = container_requests.requesting_container_uuid) as child_requests
  join containers on child_requests.container_uuid = containers.uuid
  where containers.state in ('Queued', 'Locked', 'Running')
)
select upd_container_uuid from tab;
$$;
}
  end

  def down
    ActiveRecord::Base.connection.execute "DROP FUNCTION container_priority"
    ActiveRecord::Base.connection.execute "DROP FUNCTION update_priorities"
    ActiveRecord::Base.connection.execute "DROP FUNCTION container_tree"
  end
end
