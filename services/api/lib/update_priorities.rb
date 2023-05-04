# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

def update_priorities starting_container_uuid
  ActiveRecord::Base.connection.exec_query %{
update containers set priority=computed.upd_priority from (select pri_container_uuid, upd_priority from update_priorities($1) order by pri_container_uuid) as computed
 where containers.uuid = computed.pri_container_uuid
}, 'update_priorities', [[nil, starting_container_uuid]]
end

def row_lock_for_priority_update container_uuid
  ActiveRecord::Base.connection.exec_query %{
        select 1 from containers where containers.uuid in (select pri_container_uuid from update_priorities($1)) order by containers.uuid for update
  }, 'select_for_update_priorities', [[nil, container_uuid]]
end
