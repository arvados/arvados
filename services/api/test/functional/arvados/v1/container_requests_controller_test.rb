# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

require 'test_helper'

class Arvados::V1::ContainerRequestsControllerTest < ActionController::TestCase
  test 'create with scheduling parameters' do
    authorize_with :system_user

    sp = {'partitions' => ['test1', 'test2']}
    post :create, {
      container_request: {
        command: ['echo', 'hello'],
        container_image: 'test',
        output_path: 'test',
        scheduling_parameters: sp,
      },
    }
    assert_response :success

    cr = JSON.parse(@response.body)
    assert_not_nil cr, 'Expected container request'
    assert_equal sp, cr['scheduling_parameters']
  end

  test "delete container_request and check its container's priority" do
    authorize_with :active

    # initially the container's priority is 1
    c = Container.find_by_uuid containers(:running_to_be_deleted).uuid
    assert_equal 1, c.priority

    post :destroy, id: container_requests(:running_to_be_deleted).uuid
    assert_response :success

    # now the container's priority should be set to zero
    c = Container.find_by_uuid containers(:running_to_be_deleted).uuid
    assert_equal 0, c.priority
  end
end
