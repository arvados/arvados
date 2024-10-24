# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

def row_lock_for_priority_update container_uuid
  # Locks all the containers under this container, and also any
  # immediate parent containers.  This ensures we have locked
  # everything that gets touched by either a priority update or state
  # update.
  max_retries = 6
  transaction do
    conn = ActiveRecord::Base.connection
    conn.exec_query 'SAVEPOINT row_lock_for_priority_update'
    begin
      conn.exec_query %{
        select containers.uuid from containers where containers.uuid in (
  select pri_container_uuid from container_tree($1)
UNION
  select container_requests.requesting_container_uuid from container_requests
    where container_requests.container_uuid = $1
          and container_requests.state = 'Committed'
          and container_requests.requesting_container_uuid is not NULL
)
        order by containers.uuid for update of containers
  }, 'select_for_update_priorities', [container_uuid]
    rescue ActiveRecord::Deadlocked => rn
      # bug #21540
      #
      # Despite deliberately taking the locks in uuid order,
      # reportedly this method still occasionally deadlocks with
      # another request handler that is also doing priority updates on
      # the same container tree.  This happens infrequently so we
      # don't know how to reproduce it or precisely what circumstances
      # cause it.
      #
      # However, in this situation it is safe to retry because this
      # query has no effect on the database content, its only job is
      # to acquire row locks so we can safely update the container
      # records later.

      raise if max_retries == 0
      max_retries -= 1

      # Wait random 0-10 seconds then rollback and retry
      sleep(rand(10))

      conn.exec_query 'ROLLBACK TO SAVEPOINT row_lock_for_priority_update'
      retry
    end
  end
end

def update_priorities starting_container_uuid
  # Ensure the row locks were taken in order
  row_lock_for_priority_update starting_container_uuid

  ActiveRecord::Base.connection.exec_query %{
update containers set priority=computed.upd_priority from container_tree_priorities($1) as computed
 where containers.uuid = computed.pri_container_uuid and priority != computed.upd_priority
}, 'update_priorities', [starting_container_uuid]
end
