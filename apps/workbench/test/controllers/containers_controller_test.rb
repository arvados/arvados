require 'test_helper'

class ContainersControllerTest < ActionController::TestCase
  test "visit container log" do
    use_token 'active'

    container = api_fixture('containers')['completed']

    get :show, {id: container['uuid'], tab_pane: 'Log'}, session_for(:active)
    assert_response :success

    assert_includes @response.body, "<a href=\"/collections/#{container['log']}\">Download the full log</a>"
    assert_includes @response.body, "<div class=\"collection_files_row filterable \" href=\"#{container['log']}/baz\">"
  end
end
