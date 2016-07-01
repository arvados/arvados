require 'test_helper'

class ContainersControllerTest < ActionController::TestCase
  test "visit container log" do
    use_token 'active'

    container = api_fixture('containers')['completed']

    get :show, {id: container['uuid'], tab_pane: 'Log'}, session_for(:active)
    assert_response :success

    assert_select "a", {:href=>"/collections/#{container['log']}", :text=>"Download the log"}
    assert_select "a", {:href=>"#{container['log']}/baz"}
  end
end
