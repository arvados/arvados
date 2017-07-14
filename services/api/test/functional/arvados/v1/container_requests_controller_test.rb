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
end
