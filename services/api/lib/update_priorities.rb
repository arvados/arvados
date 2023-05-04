# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

def row_lock_for_priority_update container_uuid
  ActiveRecord::Base.connection.exec_query %{
        select 1 from containers where containers.uuid in (select pri_container_uuid from container_tree($1)) order by containers.uuid for update
  }, 'select_for_update_priorities', [[nil, container_uuid]]
end

def update_priorities starting_container_uuid
  # Ensure the row locks were taken in order
  row_lock_for_priority_update starting_container_uuid

  ActiveRecord::Base.connection.exec_query %{
update containers set priority=computed.upd_priority from container_tree_priorities($1) as computed
 where containers.uuid = computed.pri_container_uuid and priority != computed.upd_priority
}, 'update_priorities', [[nil, starting_container_uuid]]
end
