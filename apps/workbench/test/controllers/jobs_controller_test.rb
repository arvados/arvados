require 'test_helper'

class JobsControllerTest < ActionController::TestCase
  test "visit jobs index page" do
    get :index, {}, session_for(:active)
    assert_response :success
  end
end
