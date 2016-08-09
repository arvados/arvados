require 'test_helper'

class WorkflowsControllerTest < ActionController::TestCase
  test "index" do
    get :index, {}, session_for(:active)
    assert_response :success
    assert_includes @response.body, 'Valid workflow with no workflow yaml'
  end
end
