# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

class PriorityUpdateFix < ActiveRecord::Migration[5.2]
  def up
    ActiveRecord::Base.connection.execute %{
CREATE OR REPLACE FUNCTION container_priority(for_container_uuid character varying, inherited bigint, inherited_from character varying) returns bigint
    LANGUAGE sql
    AS $$
/* Determine the priority of an individual container.
   The "inherited" priority comes from the path we followed from the root, the parent container
   priority hasn't been updated in the table yet but we need to behave it like it has been.
*/
select coalesce(max(case when containers.uuid = inherited_from then inherited
                         when containers.priority is not NULL then containers.priority
                         else container_requests.priority * 1125899906842624::bigint - (extract(epoch from container_requests.created_at)*1000)::bigint
                    end), 0) from
    container_requests left outer join containers on container_requests.requesting_container_uuid = containers.uuid
    where container_requests.container_uuid = for_container_uuid and
          container_requests.state = 'Committed' and
          container_requests.priority > 0 and
          container_requests.owner_uuid not in (select group_uuid from trashed_groups);
$$;
}
  end

  def down
  end
end
