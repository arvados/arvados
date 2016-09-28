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
end
