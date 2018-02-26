# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

require 'test_helper'

class ContainerRequestsControllerTest < ActionController::TestCase
  test "visit completed container request log tab" do
    use_token 'active'

    cr = api_fixture('container_requests')['completed']
    container_uuid = cr['container_uuid']
    container = Container.find(container_uuid)

    get :show, {id: cr['uuid'], tab_pane: 'Log'}, session_for(:active)
    assert_response :success

    assert_select "a", {:href=>"/collections/#{container['log']}", :text=>"Download the log"}
    assert_select "a", {:href=>"#{container['log']}/baz"}
    assert_not_includes @response.body, '<pre id="event_log_div"'
  end

  test "visit running container request log tab" do
    use_token 'active'

    cr = api_fixture('container_requests')['running']
    container_uuid = cr['container_uuid']
    container = Container.find(container_uuid)

    get :show, {id: cr['uuid'], tab_pane: 'Log'}, session_for(:active)
    assert_response :success

    assert_includes @response.body, '<pre id="event_log_div"'
    assert_select 'Download the log', false
  end

  test "completed container request offers re-run option" do
    use_token 'active'

    uuid = api_fixture('container_requests')['completed']['uuid']

    get :show, {id: uuid}, session_for(:active)
    assert_response :success

    assert_includes @response.body, "action=\"/container_requests/#{uuid}/copy\""
  end

  test "cancel request for queued container" do
    cr_fixture = api_fixture('container_requests')['queued']
    post :cancel, {id: cr_fixture['uuid']}, session_for(:active)
    assert_response 302

    use_token 'active'
    cr = ContainerRequest.find(cr_fixture['uuid'])
    assert_equal 'Final', cr.state
    assert_equal 0, cr.priority
    c = Container.find(cr_fixture['container_uuid'])
    assert_equal 'Queued', c.state
    assert_equal 0, c.priority
  end

  [
    ['completed', false, false],
    ['completed', true, false],
    ['completed-older', false, true],
    ['completed-older', true, true],
  ].each do |cr_fixture, reuse_enabled, uses_acr|
    test "container request #{uses_acr ? '' : 'not'} using arvados-cwl-runner copy #{reuse_enabled ? 'with' : 'without'} reuse enabled" do
      completed_cr = api_fixture('container_requests')[cr_fixture]
      # Set up post request params
      copy_params = {id: completed_cr['uuid']}
      if reuse_enabled
        copy_params.merge!({use_existing: true})
      end
      post(:copy, copy_params, session_for(:active))
      assert_response 302
      copied_cr = assigns(:object)
      assert_not_nil copied_cr
      assert_equal 'Uncommitted', copied_cr[:state]
      assert_equal "Copy of #{completed_cr['name']}", copied_cr['name']
      assert_equal completed_cr['cmd'], copied_cr['cmd']
      assert_equal completed_cr['runtime_constraints']['ram'], copied_cr['runtime_constraints'][:ram]
      if reuse_enabled
        assert copied_cr[:use_existing]
      else
        refute copied_cr[:use_existing]
      end
      # If the CR's command is arvados-cwl-runner, the appropriate flag should
      # be passed to it
      if uses_acr
        if reuse_enabled
          # arvados-cwl-runner's default behavior is to enable reuse
          assert_includes copied_cr['command'], 'arvados-cwl-runner'
          assert_not_includes copied_cr['command'], '--disable-reuse'
        else
          assert_includes copied_cr['command'], 'arvados-cwl-runner'
          assert_includes copied_cr['command'], '--disable-reuse'
          assert_not_includes copied_cr['command'], '--enable-reuse'
        end
      else
        # If no arvados-cwl-runner is being used, the command should be left alone
        assert_equal completed_cr['command'], copied_cr['command']
      end
    end
  end

  [
    ['completed', true],
    ['running', true],
    ['queued', true],
    ['uncommitted', false],
  ].each do |cr_fixture, should_show|
    test "provenance tab should #{should_show ? '' : 'not'} be shown on #{cr_fixture} container requests" do
      cr = api_fixture('container_requests')[cr_fixture]
      assert_not_nil cr
      get(:show,
          {id: cr['uuid']},
          session_for(:active))
      assert_response :success
      if should_show
        assert_includes @response.body, "href=\"#Provenance\""
      else
        assert_not_includes @response.body, "href=\"#Provenance\""
      end
    end
  end

  test "container request display" do
    use_token 'active'

    cr = api_fixture('container_requests')['completed_with_input_mounts']

    get :show, {id: cr['uuid']}, session_for(:active)
    assert_response :success

    assert_match /hello/, @response.body
    assert_includes @response.body, "href=\"\/collections/fa7aeb5140e2848d39b416daeef4ffc5+45/foo" # mount input1
    assert_includes @response.body, "href=\"\/collections/fa7aeb5140e2848d39b416daeef4ffc5+45/bar" # mount input2
    assert_includes @response.body, "href=\"\/collections/1fd08fc162a5c6413070a8bd0bffc818+150" # mount workflow
    assert_includes @response.body, "href=\"#Log\""
    assert_includes @response.body, "href=\"#Provenance\""
  end
end
