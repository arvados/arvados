# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

require 'test_helper'

class WorkflowsControllerTest < ActionController::TestCase
  test "index" do
    get :index, {}, session_for(:active)
    assert_response :success
    assert_includes @response.body, 'Valid workflow with no definition yaml'
  end

  test "show" do
    use_token 'active'

    wf = api_fixture('workflows')['workflow_with_input_specifications']

    get :show, {id: wf['uuid']}, session_for(:active)
    assert_response :success

    assert_includes @response.body, "a short label for this parameter (optional)"
    assert_includes @response.body, "href=\"#Advanced\""
  end
end
