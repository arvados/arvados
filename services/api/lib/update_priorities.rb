# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

def row_lock_for_priority_update container_uuid
  # Locks all the containers under this container, and also any
  # immediate parent containers.  This ensures we have locked
  # everything that gets touched by either a priority update or state
  # update.
  # This method assumes we are already in a transaction.
  ActiveRecord::Base.connection.exec_query %{
        select containers.id from containers where containers.uuid in (
  select pri_container_uuid from container_tree($1)
UNION
  select container_requests.requesting_container_uuid from container_requests
    where container_requests.container_uuid = $1
          and container_requests.state = 'Committed'
          and container_requests.requesting_container_uuid is not NULL
)
        order by containers.id for update of containers
  }, 'select_for_update_priorities', [container_uuid]
end

def update_priorities starting_container_uuid
  Container.transaction do
    # Ensure the row locks were taken in order
    row_lock_for_priority_update starting_container_uuid

    ActiveRecord::Base.connection.exec_query %{
update containers set priority=computed.upd_priority from container_tree_priorities($1) as computed
 where containers.uuid = computed.pri_container_uuid and priority != computed.upd_priority
}, 'update_priorities', [starting_container_uuid]
  end
end
