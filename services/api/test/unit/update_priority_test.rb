# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

require 'test_helper'

class UpdatePriorityTest < ActiveSupport::TestCase
  test 'priority 0 but should be >0' do
    uuid = containers(:running).uuid
    ActiveRecord::Base.connection.exec_query('UPDATE containers SET priority=0 WHERE uuid=$1', 'test-setup', [[nil, uuid]])
    assert_equal 0, Container.find_by_uuid(uuid).priority
    UpdatePriority.update_priority(nolock: true)
    assert_operator 0, :<, Container.find_by_uuid(uuid).priority

    uuid = containers(:queued).uuid
    ActiveRecord::Base.connection.exec_query('UPDATE containers SET priority=0 WHERE uuid=$1', 'test-setup', [[nil, uuid]])
    assert_equal 0, Container.find_by_uuid(uuid).priority
    UpdatePriority.update_priority(nolock: true)
    assert_operator 0, :<, Container.find_by_uuid(uuid).priority
  end

  test 'priority>0 but should be 0' do
    uuid = containers(:running).uuid
    ActiveRecord::Base.connection.exec_query('DELETE FROM container_requests WHERE container_uuid=$1', 'test-setup', [[nil, uuid]])
    assert_operator 0, :<, Container.find_by_uuid(uuid).priority
    UpdatePriority.update_priority(nolock: true)
    assert_equal 0, Container.find_by_uuid(uuid).priority
  end
end
