# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

class AddQueueIndexToContainers < ActiveRecord::Migration
  def up
    ActiveRecord::Base.connection.execute 'CREATE INDEX index_containers_on_queued_state on containers (state, (priority > 0))'
  end
  def down
    ActiveRecord::Base.connection.execute 'DROP INDEX index_containers_on_queued_state'
  end
end
