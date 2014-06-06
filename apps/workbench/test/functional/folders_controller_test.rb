require 'test_helper'

class FoldersControllerTest < ActionController::TestCase
  test "inactive user is asked to sign user agreements on front page" do
    get :index, {}, session_for(:inactive)
    assert_response :success
    assert_not_empty assigns(:required_user_agreements),
    "Inactive user did not have required_user_agreements"
    assert_template 'user_agreements/index',
    "Inactive user was not presented with a user agreement at the front page"
  end
end
