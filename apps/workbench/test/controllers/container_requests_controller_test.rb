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
    assert_not_includes @response.body, '<div id="event_log_div"'
  end

  test "visit running container request log tab" do
    use_token 'active'

    cr = api_fixture('container_requests')['running']
    container_uuid = cr['container_uuid']
    container = Container.find(container_uuid)

    get :show, {id: cr['uuid'], tab_pane: 'Log'}, session_for(:active)
    assert_response :success

    assert_includes @response.body, '<div id="event_log_div"'
    assert_select 'Download the log', false
  end

  test "completed container request offers re-run option" do
    use_token 'active'

    uuid = api_fixture('container_requests')['completed']['uuid']

    get :show, {id: uuid}, session_for(:active)
    assert_response :success

   assert_includes @response.body, "action=\"/container_requests/#{uuid}/copy\""
  end

  test "container request copy" do
    completed_cr = api_fixture('container_requests')['completed']
    post(:copy,
         {
           id: completed_cr['uuid']
         },
         session_for(:active))
    assert_response 302
    copied_cr = assigns(:object)
    assert_not_nil copied_cr
    assert_equal 'Uncommitted', copied_cr[:state]
    assert_equal "Copy of #{completed_cr['name']}", copied_cr['name']
    assert_equal completed_cr['cmd'], copied_cr['cmd']
    assert_equal completed_cr['runtime_constraints']['ram'], copied_cr['runtime_constraints'][:ram]
    refute copied_cr[:use_existing]
  end

  test "container request copy with reuse enabled" do
    completed_cr = api_fixture('container_requests')['completed']
    post(:copy,
         {
           id: completed_cr['uuid'],
           use_existing: true,
         },
         session_for(:active))
    assert_response 302
    copied_cr = assigns(:object)
    assert_not_nil copied_cr
    assert copied_cr['use_existing']
  end
end
