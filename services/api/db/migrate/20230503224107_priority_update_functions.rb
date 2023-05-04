# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

class PriorityUpdateFunctions < ActiveRecord::Migration[5.2]
  def up
    ActiveRecord::Base.connection.execute %{
CREATE OR REPLACE FUNCTION container_priority(for_container_uuid character varying, inherited bigint) returns bigint
    LANGUAGE sql
    AS $$
select coalesce(max(case when container_requests.priority = 0 then 0
                         when containers.priority is not NULL then greatest(containers.priority, inherited)
                         else container_requests.priority * 1125899906842624::bigint - (extract(epoch from container_requests.created_at)*1000)::bigint
                    end), 0) from
    container_requests left outer join containers on container_requests.requesting_container_uuid = containers.uuid
    where container_requests.container_uuid = for_container_uuid and container_requests.state = 'Committed' and container_requests.priority > 0;
$$;
}

    ActiveRecord::Base.connection.execute %{
CREATE OR REPLACE FUNCTION update_priorities(for_container_uuid character varying) returns table (pri_container_uuid character varying, priority bigint)
    LANGUAGE sql
    AS $$
with recursive tab(upd_container_uuid, upd_priority) as (
  select for_container_uuid, container_priority(for_container_uuid, 0)
union
  select containers.uuid, container_priority(containers.uuid, child_requests.upd_priority)
  from (tab join container_requests on tab.upd_container_uuid = container_requests.requesting_container_uuid) as child_requests
  join containers on child_requests.container_uuid = containers.uuid
  where containers.state in ('Queued', 'Locked', 'Running')
)
select upd_container_uuid, upd_priority from tab;
$$;
}
  end

  def down
    ActiveRecord::Base.connection.execute "DROP FUNCTION container_priority"
    ActiveRecord::Base.connection.execute "DROP FUNCTION update_priorities"
  end
end
