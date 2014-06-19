require 'test_helper'

class ProjectsControllerTest < ActionController::TestCase
  setup do
    @anonymous_token = Rails.configuration.anonymous_user_token
  end

  teardown do
    Rails.configuration.anonymous_user_token = @anonymous_token
  end

  test "inactive user is asked to sign user agreements on front page when anonymous user token is not configured" do
    Rails.configuration.anonymous_user_token = false
    get :index, {}, session_for(:inactive)
    assert_response :success
    assert_not_empty assigns(:required_user_agreements),
    "Inactive user did not have required_user_agreements"
    assert_template 'user_agreements/index',
    "Inactive user was not presented with a user agreement at the front page"
  end

  test "inactive user is asked to sign user agreements on front page" do
    get :index, {}, session_for(:inactive)
    assert_response :success
    if !@anonymous_token
      assert_not_empty assigns(:required_user_agreements),
      "Inactive user did not have required_user_agreements"
      assert_template 'user_agreements/index',
      "Inactive user was not presented with a user agreement at the front page"
    else
      assert_nil assigns(:required_user_agreements),
      "Inactive user did not have required_user_agreements"
    end
  end

end
