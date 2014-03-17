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

  test "create new user with user as input" do
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

	test "create user with user, vm and repo as input" do
    authorize_with :admin

    post :create, {
      repo_name: 'test_repo',
			vm_uuid: 'abcdefg',
      user: {
		    uuid: "is_this_correct",		    
				first_name: "in_create_test_first_name",
		    last_name: "test_last_name",
				email: "test@abc.com"
      }
    }
    assert_response :success
    created = JSON.parse(@response.body)
    assert_equal 'in_create_test_first_name', created['first_name']
    assert_not_nil created['uuid'], 'expected non-null uuid for the newly created user'
		assert_equal 'is_this_correct', created['uuid']
    assert_not_nil created['email'], 'since email was given, expected non-nil email'
    assert_nil created['identity_url'], 'even though email is provided, expected no identity_url since users_controller only creates user' 
	end

	test "create user with user_param, vm and repo as input" do
    authorize_with :admin

    post :create, {
      user_param: 'not_an_existing_uuid_and_not_email_format',
      repo_name: 'test_repo',
			vm_uuid: 'abcdefg',
      user: {}
    }

    response_body = JSON.parse(@response.body)
    response_errors = response_body['errors']
		assert_not_nil response_errors, 'Expected error in response'
		incorrectly_formatted = response_errors.first.include?('ArgumentError: User param is not of valid email format')
		assert incorrectly_formatted, 'Expected not valid email format error'
	end

	test "create user with existing uuid user_param, vm and repo as input" do
		authorize_with :inactive
    get :current
    assert_response :success
    inactive_user = JSON.parse(@response.body)
		
    authorize_with :admin

		# it would be desirable to use inactive_user['uuid'] instead of hard coding user_param
    post :create, {
      user_param: 'zzzzz-tpzed-x9kqpd79egh49c7',
      repo_name: 'test_repo',
			vm_uuid: 'abcdefg',
      user: {}
    }

    assert_response :success
    response_object = JSON.parse(@response.body)
    assert_not_nil response_object['uuid'], 'expected non-null uuid for the newly created user'
		assert_equal inactive_user['uuid'], response_object['uuid']
    assert_equal inactive_user['email'], response_object['email'], 'expecting inactive user email'
	end

	test "create user with valid email user_param, vm and repo as input" do
    authorize_with :admin

    post :create, {
      user_param: 'abc@xyz.com',
      repo_name: 'test_repo',
			vm_uuid: 'abcdefg',
      user: {}
    }

    assert_response :success
    response_object = JSON.parse(@response.body)
    assert_not_nil response_object['uuid'], 'expected non-null uuid for the newly created user'
    assert_equal response_object['email'], 'abc@xyz.com', 'expecting given email'
	end

	test "create user with valid email user_param, no vm and repo as input" do
    authorize_with :admin

    post :create, {
      user_param: 'abc@xyz.com',
      user: {}
    }

    assert_response :success		
    response_object = JSON.parse(@response.body)
    assert_not_nil response_object['uuid'], 'expected non-null uuid for the newly created user'
    assert_equal response_object['email'], 'abc@xyz.com', 'expecting given email'
	end

	test "create user with valid email user_param, vm and repo as input with opt.n" do
    authorize_with :admin

    post :create, {
      user_param: 'abc@xyz.com',
      repo_name: 'test_repo',
			vm_uuid: 'abcdefg',
			just_probe: 'true',
      user: {}
    }

    assert_response :success
    response_object = JSON.parse(@response.body)
    assert_nil response_object['uuid'], 'expected null uuid since no object created due to just probe'
    assert_nil response_object['email'], 'expecting null email'
	end

	# in progress
	test "create user twice and check links are not recreated" do
    authorize_with :admin

    post :create, {
      user_param: 'abc@xyz.com',
      repo_name: 'test_repo',
			vm_uuid: 'abcdefg',
      user: {}
    }

    assert_response :success
    response_object = JSON.parse(@response.body)
    assert_not_nil response_object['uuid'], 'expected non-null uuid for the newly created user'
    assert_equal response_object['email'], 'abc@xyz.com', 'expecting given email'

		# create again
    post :create, {
      user_param: 'abc@xyz.com',
      repo_name: 'test_repo',
			vm_uuid: 'abcdefg',
      user: {}
    }

    assert_response :success
    response_object2 = JSON.parse(@response.body)
    assert_equal response_object['uuid'], response_object2['uuid'], 'expected same uuid as first create operation'
    assert_equal response_object['email'], 'abc@xyz.com', 'expecting given email'

		# check links are not recreated
	end

	test "create user with openid_prefix" do
    authorize_with :admin

    post :create, {
      repo_name: 'test_repo',
			vm_uuid: 'abcdefg',
      user: {
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
