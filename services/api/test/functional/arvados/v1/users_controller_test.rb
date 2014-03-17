require 'test_helper'

class Arvados::V1::UsersControllerTest < ActionController::TestCase

  test "activate a user after signing UA" do
    authorize_with :inactive_but_signed_user_agreement
    get :current
    assert_response :success
    me = JSON.parse(@response.body)
    post :activate, uuid: me['uuid']
    assert_response :success
    assert_not_nil assigns(:object)
    me = JSON.parse(@response.body)
    assert_equal true, me['is_active']
  end

  test "refuse to activate a user before signing UA" do
    authorize_with :inactive
    get :current
    assert_response :success
    me = JSON.parse(@response.body)
    post :activate, uuid: me['uuid']
    assert_response 403
    get :current
    assert_response :success
    me = JSON.parse(@response.body)
    assert_equal false, me['is_active']
  end

  test "activate an already-active user" do
    authorize_with :active
    get :current
    assert_response :success
    me = JSON.parse(@response.body)
    post :activate, uuid: me['uuid']
    assert_response :success
    me = JSON.parse(@response.body)
    assert_equal true, me['is_active']
  end

  test "create new user" do
    authorize_with :admin
    post :create, user: {
      first_name: "test_first_name",
      last_name: "test_last_name",
			email: "test@abc.com"
    }
    assert_response :success
    created = JSON.parse(@response.body)
    assert_equal 'test_first_name', created['first_name']
    assert_not_nil created['uuid'], 'expected non-null uuid for the newly created user'
    assert_not_nil created['email'], 'since email was given, expected non-nil email'
    assert_nil created['identity_url'], 'even though email is provided, expected no identity_url since users_controller only creates user at this time'
  end

	test "create user with vm and repo" do
    authorize_with :admin

    post :create, {
      repo_name: 'test_repo',
			vm_uuid: 'abcdefg',
      user: {
		    uuid: "shouldnotbeused",		    
				first_name: "in_create_test_first_name",
		    last_name: "test_last_name",
				email: "test@abc.com"
      }
    }
    assert_response :success
    created = JSON.parse(@response.body)
    assert_equal 'in_create_test_first_name', created['first_name']
    assert_not_nil created['uuid'], 'expected non-null uuid for the newly created user'
    assert_not_nil created['email'], 'since email was given, expected non-nil email'
    assert_nil created['identity_url'], 'even though email is provided, expected no identity_url since users_controller only creates user' 
	end

end
