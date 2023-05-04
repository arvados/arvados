# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

def update_priorities starting_container_uuid
  ActiveRecord::Base.connection.exec_query %{
update containers set priority=computed.priority from (select pri_container_uuid, priority from update_priorities($1) order by pri_container_uuid) as computed
 where containers.uuid = computed.pri_container_uuid
}, 'update_priorities', [[nil, starting_container_uuid]]
end
