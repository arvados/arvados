require 'test_helper'

class ContainerRequestsControllerTest < ActionController::TestCase
  test "visit container_request log" do
    use_token 'active'

    cr = api_fixture('container_requests')['completed']
    container_uuid = cr['container_uuid']
    container = Container.find(container_uuid)

    get :show, {id: cr['uuid'], tab_pane: 'Log'}, session_for(:active)
    assert_response :success

    assert_select "a", {:href=>"/collections/#{container['log']}", :text=>"Download the log"}
    assert_select "a", {:href=>"#{container['log']}/baz"}
  end
end
