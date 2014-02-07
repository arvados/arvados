require 'test_helper'

class Arvados::V1::RepositoriesControllerTest < ActionController::TestCase
  test "should get_all_logins with admin token" do
    authorize_with :admin
    get :get_all_permissions
    assert_response :success
  end

  test "should get_all_logins with non-admin token" do
    authorize_with :active
    get :get_all_permissions
    assert_response 403
  end
end
