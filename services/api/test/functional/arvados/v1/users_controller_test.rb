require 'test_helper'

class Arvados::V1::UsersControllerTest < ActionController::TestCase

	setup do
		@all_users_at_start = User.all
		@all_groups_at_start = Group.all
		@all_links_at_start = Link.all

		@vm_uuid = virtual_machines(:testvm).uuid
	end

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
		repo_name = 'test_repo'

    post :create, {
      repo_name: repo_name,
			vm_uuid: 'no_such_vm',
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
    assert_nil created['identity_url'], 'even though email isprovided, expected no identity_url since users_controller only creates user' 

		# since no such vm exists, expect only three new links: oid_login_perm, repo link and link add user to 'All users' group
		verify_num_links @all_links_at_start, 3

		verify_link_exists_for_type 'User', 'permission', 'can_login', created['uuid'], created['email'], 'arvados#user', false

		verify_link_exists_for_type 'Repository', 'permission', 'can_write', repo_name, created['uuid'], 'arvados#repository', true

		verify_link_exists_for_type 'Group', 'permission', 'can_read', 'All users', created['uuid'], 'arvados#group', true
	end

	test "create user with user_param, vm and repo as input" do
    authorize_with :admin

    post :create, {
      user_param: 'not_an_existing_uuid_and_not_email_format',
      repo_name: 'test_repo',
			vm_uuid: 'no_such_vm',
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

    post :create, {
      user_param: inactive_user['uuid'],
      repo_name: 'test_repo',
			vm_uuid: 'no_such_vm',
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
			vm_uuid: 'no_such_vm',
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

	test "create user with user_param and user which will be ignored" do
    authorize_with :admin

    post :create, {
      user_param: 'abc@xyz.com',
      repo_name: 'test_repo',
			vm_uuid: 'no_such_vm',
      user: {
				email: 'will_be_ignored@xyz.om'
			}
    }

    assert_response :success
    response_object = JSON.parse(@response.body)
    assert_not_nil response_object['uuid'], 'expected non-null uuid for the newly created user'
    assert_equal response_object['email'], 'abc@xyz.com', 'expecting user_param as email'
	end

	test "create user with valid email user_param, vm and repo as input with opt.n" do
    authorize_with :admin

    post :create, {
      user_param: 'abc@xyz.com',
      repo_name: 'test_repo',
			vm_uuid: 'no_such_vm',
			just_probe: 'true',
      user: {}
    }

    assert_response :success
    response_object = JSON.parse(@response.body)
    assert_nil response_object['uuid'], 'expected null uuid since no object created due to just probe'
    assert_nil response_object['email'], 'expecting null email'
	end

	test "create user twice and check links are not recreated" do
    authorize_with :admin

    post :create, {
      user_param: 'abc@xyz.com',
      repo_name: 'test_repo',
			vm_uuid: 'no_such_vm',
      user: {}
    }

    assert_response :success
    response_object = JSON.parse(@response.body)
    assert_not_nil response_object['uuid'], 'expected non-null uuid for the newly created user'
    assert_equal response_object['email'], 'abc@xyz.com', 'expecting given email'
		verify_num_links @all_links_at_start, 3		# openid, group, and repo links. no vm link

		# create again
    post :create, {
      user_param: 'abc@xyz.com',
      repo_name: 'test_repo',
			vm_uuid: 'no_such_vm',
      user: {}
    }

    assert_response :success
    response_object2 = JSON.parse(@response.body)
    assert_equal response_object['uuid'], response_object2['uuid'], 'expected same uuid as first create operation'
    assert_equal response_object['email'], 'abc@xyz.com', 'expecting given email'
		verify_num_links @all_links_at_start, 3		# openid, group, and repo links. no vm link
	end

	test "create user with openid prefix" do
    authorize_with :admin

    post :create, {
      repo_name: 'test_repo',
			vm_uuid: 'no_such_vm',
			openid_prefix: 'http://www.xyz.com/account',
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

		# verify links
		# expect 3 new links: oid_login_perm, repo link, and link add user to 'All users' group. No vm link since the vm_uuid passed in is not in system
		verify_num_links @all_links_at_start, 3

		verify_link_exists_for_type 'User', 'permission', 'can_login', created['uuid'], created['email'], 'arvados#user', false

		verify_link_exists_for_type 'Repository', 'permission', 'can_write', 'test_repo', created['uuid'], 'arvados#repository', true

		verify_link_exists_for_type 'Group', 'permission', 'can_read', 'All users', created['uuid'], 'arvados#group', true
	end

	test "create user with user, vm and repo and verify links" do
    authorize_with :admin

    post :create, {
      repo_name: 'test_repo',
			vm_uuid: @vm_uuid,
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

		# expect 4 new links: oid_login_perm, repo link, vm link and link add user to 'All users' group. 
		verify_num_links @all_links_at_start, 4

		verify_link_exists_for_type 'User', 'permission', 'can_login', created['uuid'], created['email'], 'arvados#user', false

		verify_link_exists_for_type 'Repository', 'permission', 'can_write', 'test_repo', created['uuid'], 'arvados#repository', true

		verify_link_exists_for_type 'Group', 'permission', 'can_read', 'All users', created['uuid'], 'arvados#group', true

		verify_link_exists_for_type 'VirtualMachine', 'permission', 'can_login', @vm_uuid, created['uuid'], 'arvados#virtualMachine', false
	end

	def verify_num_links (original_links, expected_num_additional_links)
		links_now = Link.all
		assert_equal original_links.size+expected_num_additional_links, Link.all.size, 
							"Expected #{expected_num_additional_links.inspect} more links"
	end

	def verify_link_exists_for_type(class_name, link_class, link_name, head_uuid, tail_uuid, head_kind, fetch_object)
		if fetch_object
			object = Object.const_get(class_name).where(name: head_uuid)
			assert [] != object, "expected a #{class_name.inspect} with the name #{head_uuid.inspect}"
			head_uuid = object.first[:uuid]
		end

		links = Link.where(link_class: link_class,
              				 name: link_name,
             					 tail_uuid: tail_uuid,
             					 head_uuid: head_uuid,
				               head_kind: head_kind)
		assert links.size > 0, "expected one or more links with the given criteria #{class_name} with #{head_uuid}"
	end

end
