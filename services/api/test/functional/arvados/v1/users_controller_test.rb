require 'test_helper'
require 'helpers/users_test_helper'

class Arvados::V1::UsersControllerTest < ActionController::TestCase
  include CurrentApiClient
  include UsersTestHelper

  setup do
    @all_links_at_start = Link.all
    @vm_uuid = virtual_machines(:testvm).uuid
  end

  test "activate a user after signing UA" do
    authorize_with :inactive_but_signed_user_agreement
    post :activate, id: users(:inactive_but_signed_user_agreement).uuid
    assert_response :success
    assert_not_nil assigns(:object)
    me = JSON.parse(@response.body)
    assert_equal true, me['is_active']
  end

  test "refuse to activate a user before signing UA" do
    act_as_system_user do
    required_uuids = Link.where("owner_uuid = ? and link_class = ? and name = ? and tail_uuid = ? and head_uuid like ?",
                                system_user_uuid,
                                'signature',
                                'require',
                                system_user_uuid,
                                Collection.uuid_like_pattern).
      collect(&:head_uuid)

      assert required_uuids.length > 0

      signed_uuids = Link.where(owner_uuid: system_user_uuid,
                                link_class: 'signature',
                                name: 'click',
                                tail_uuid: users(:inactive).uuid,
                                head_uuid: required_uuids).
                          collect(&:head_uuid)

      assert_equal 0, signed_uuids.length
    end

    authorize_with :inactive
    assert_equal false, users(:inactive).is_active

    post :activate, id: users(:inactive).uuid
    assert_response 403

    resp = json_response
    assert resp['errors'].first.include? 'Cannot activate without user agreements'
    assert_nil resp['is_active']
  end

  test "activate an already-active user" do
    authorize_with :active
    post :activate, id: users(:active).uuid
    assert_response :success
    me = JSON.parse(@response.body)
    assert_equal true, me['is_active']
  end

  test "respond 401 if given token exists but user record is missing" do
    authorize_with :valid_token_deleted_user
    get :current, {format: :json}
    assert_response 401
  end

  test "create new user with user as input" do
    authorize_with :admin
    post :create, user: {
      first_name: "test_first_name",
      last_name: "test_last_name",
      email: "foo@example.com"
    }
    assert_response :success
    created = JSON.parse(@response.body)
    assert_equal 'test_first_name', created['first_name']
    assert_not_nil created['uuid'], 'expected uuid for the newly created user'
    assert_not_nil created['email'], 'expected non-nil email'
    assert_nil created['identity_url'], 'expected no identity_url'
  end

  test "create user with user, vm and repo as input" do
    authorize_with :admin
    repo_name = 'usertestrepo'

    post :setup, {
      repo_name: repo_name,
      openid_prefix: 'https://www.google.com/accounts/o8/id',
      user: {
        uuid: 'zzzzz-tpzed-abcdefghijklmno',
        first_name: "in_create_test_first_name",
        last_name: "test_last_name",
        email: "foo@example.com"
      }
    }
    assert_response :success
    response_items = JSON.parse(@response.body)['items']

    created = find_obj_in_resp response_items, 'User', nil

    assert_equal 'in_create_test_first_name', created['first_name']
    assert_not_nil created['uuid'], 'expected non-null uuid for the new user'
    assert_equal 'zzzzz-tpzed-abcdefghijklmno', created['uuid']
    assert_not_nil created['email'], 'expected non-nil email'
    assert_nil created['identity_url'], 'expected no identity_url'

    # arvados#user, repo link and link add user to 'All users' group
    verify_num_links @all_links_at_start, 4

    verify_link response_items, 'arvados#user', true, 'permission', 'can_login',
        created['uuid'], created['email'], 'arvados#user', false, 'User'

    verify_link response_items, 'arvados#repository', true, 'permission', 'can_manage',
        "foo/#{repo_name}", created['uuid'], 'arvados#repository', true, 'Repository'

    verify_link response_items, 'arvados#group', true, 'permission', 'can_read',
        'All users', created['uuid'], 'arvados#group', true, 'Group'

    verify_link response_items, 'arvados#virtualMachine', false, 'permission', 'can_login',
        nil, created['uuid'], 'arvados#virtualMachine', false, 'VirtualMachine'

    verify_system_group_permission_link_for created['uuid']
  end

  test "setup user with bogus uuid and expect error" do
    authorize_with :admin

    post :setup, {
      uuid: 'bogus_uuid',
      repo_name: 'usertestrepo',
      vm_uuid: @vm_uuid
    }
    response_body = JSON.parse(@response.body)
    response_errors = response_body['errors']
    assert_not_nil response_errors, 'Expected error in response'
    assert (response_errors.first.include? 'Path not found'), 'Expected 404'
  end

  test "setup user with bogus uuid in user and expect error" do
    authorize_with :admin

    post :setup, {
      user: {uuid: 'bogus_uuid'},
      repo_name: 'usertestrepo',
      vm_uuid: @vm_uuid,
      openid_prefix: 'https://www.google.com/accounts/o8/id'
    }
    response_body = JSON.parse(@response.body)
    response_errors = response_body['errors']
    assert_not_nil response_errors, 'Expected error in response'
    assert (response_errors.first.include? 'ArgumentError: Require user email'),
      'Expected RuntimeError'
  end

  test "setup user with no uuid and user, expect error" do
    authorize_with :admin

    post :setup, {
      repo_name: 'usertestrepo',
      vm_uuid: @vm_uuid,
      openid_prefix: 'https://www.google.com/accounts/o8/id'
    }
    response_body = JSON.parse(@response.body)
    response_errors = response_body['errors']
    assert_not_nil response_errors, 'Expected error in response'
    assert (response_errors.first.include? 'Required uuid or user'),
        'Expected ArgumentError'
  end

  test "setup user with no uuid and email, expect error" do
    authorize_with :admin

    post :setup, {
      user: {},
      repo_name: 'usertestrepo',
      vm_uuid: @vm_uuid,
      openid_prefix: 'https://www.google.com/accounts/o8/id'
    }
    response_body = JSON.parse(@response.body)
    response_errors = response_body['errors']
    assert_not_nil response_errors, 'Expected error in response'
    assert (response_errors.first.include? '<ArgumentError: Require user email'),
        'Expected ArgumentError'
  end

  test "invoke setup with existing uuid, vm and repo and verify links" do
    authorize_with :admin
    inactive_user = users(:inactive)

    post :setup, {
      uuid: users(:inactive).uuid,
      repo_name: 'usertestrepo',
      vm_uuid: @vm_uuid
    }

    assert_response :success

    response_items = JSON.parse(@response.body)['items']
    resp_obj = find_obj_in_resp response_items, 'User', nil

    assert_not_nil resp_obj['uuid'], 'expected uuid for the new user'
    assert_equal inactive_user['uuid'], resp_obj['uuid']
    assert_equal inactive_user['email'], resp_obj['email'],
        'expecting inactive user email'

    # expect repo and vm links
    verify_link response_items, 'arvados#repository', true, 'permission', 'can_manage',
        'inactiveuser/usertestrepo', resp_obj['uuid'], 'arvados#repository', true, 'Repository'

    verify_link response_items, 'arvados#virtualMachine', true, 'permission', 'can_login',
        @vm_uuid, resp_obj['uuid'], 'arvados#virtualMachine', false, 'VirtualMachine'
  end

  test "invoke setup with existing uuid in user, verify response" do
    authorize_with :admin
    inactive_user = users(:inactive)

    post :setup, {
      user: {uuid: inactive_user['uuid']},
      openid_prefix: 'https://www.google.com/accounts/o8/id'
    }

    assert_response :success

    response_items = JSON.parse(@response.body)['items']
    resp_obj = find_obj_in_resp response_items, 'User', nil

    assert_not_nil resp_obj['uuid'], 'expected uuid for the new user'
    assert_equal inactive_user['uuid'], resp_obj['uuid']
    assert_equal inactive_user['email'], resp_obj['email'],
        'expecting inactive user email'
  end

  test "invoke setup with existing uuid but different email, expect original email" do
    authorize_with :admin
    inactive_user = users(:inactive)

    post :setup, {
      uuid: inactive_user['uuid'],
      user: {email: 'junk_email'}
    }

    assert_response :success

    response_items = JSON.parse(@response.body)['items']
    resp_obj = find_obj_in_resp response_items, 'User', nil

    assert_not_nil resp_obj['uuid'], 'expected uuid for the new user'
    assert_equal inactive_user['uuid'], resp_obj['uuid']
    assert_equal inactive_user['email'], resp_obj['email'],
        'expecting inactive user email'
  end

  test "setup user with valid email and repo as input" do
    authorize_with :admin

    post :setup, {
      repo_name: 'usertestrepo',
      user: {email: 'foo@example.com'},
      openid_prefix: 'https://www.google.com/accounts/o8/id'
    }

    assert_response :success
    response_items = JSON.parse(@response.body)['items']
    response_object = find_obj_in_resp response_items, 'User', nil
    assert_not_nil response_object['uuid'], 'expected uuid for the new user'
    assert_equal response_object['email'], 'foo@example.com', 'expected given email'

    # four extra links; system_group, login, group and repo perms
    verify_num_links @all_links_at_start, 4
  end

  test "setup user with fake vm and expect error" do
    authorize_with :admin

    post :setup, {
      repo_name: 'usertestrepo',
      vm_uuid: 'no_such_vm',
      user: {email: 'foo@example.com'},
      openid_prefix: 'https://www.google.com/accounts/o8/id'
    }

    response_body = JSON.parse(@response.body)
    response_errors = response_body['errors']
    assert_not_nil response_errors, 'Expected error in response'
    assert (response_errors.first.include? "No vm found for no_such_vm"),
          'Expected RuntimeError: No vm found for no_such_vm'
  end

  test "setup user with valid email, repo and real vm as input" do
    authorize_with :admin

    post :setup, {
      repo_name: 'usertestrepo',
      openid_prefix: 'https://www.google.com/accounts/o8/id',
      vm_uuid: @vm_uuid,
      user: {email: 'foo@example.com'}
    }

    assert_response :success
    response_items = JSON.parse(@response.body)['items']
    response_object = find_obj_in_resp response_items, 'User', nil
    assert_not_nil response_object['uuid'], 'expected uuid for the new user'
    assert_equal response_object['email'], 'foo@example.com', 'expected given email'

    # five extra links; system_group, login, group, vm, repo
    verify_num_links @all_links_at_start, 5
  end

  test "setup user with valid email, no vm and no repo as input" do
    authorize_with :admin

    post :setup, {
      user: {email: 'foo@example.com'},
      openid_prefix: 'https://www.google.com/accounts/o8/id'
    }

    assert_response :success
    response_items = JSON.parse(@response.body)['items']
    response_object = find_obj_in_resp response_items, 'User', nil
    assert_not_nil response_object['uuid'], 'expected uuid for new user'
    assert_equal response_object['email'], 'foo@example.com', 'expected given email'

    # three extra links; system_group, login, and group
    verify_num_links @all_links_at_start, 3

    verify_link response_items, 'arvados#user', true, 'permission', 'can_login',
        response_object['uuid'], response_object['email'], 'arvados#user', false, 'User'

    verify_link response_items, 'arvados#group', true, 'permission', 'can_read',
        'All users', response_object['uuid'], 'arvados#group', true, 'Group'

    verify_link response_items, 'arvados#repository', false, 'permission', 'can_manage',
        'foo/usertestrepo', response_object['uuid'], 'arvados#repository', true, 'Repository'

    verify_link response_items, 'arvados#virtualMachine', false, 'permission', 'can_login',
        nil, response_object['uuid'], 'arvados#virtualMachine', false, 'VirtualMachine'
  end

  test "setup user with email, first name, repo name and vm uuid" do
    authorize_with :admin

    post :setup, {
      openid_prefix: 'https://www.google.com/accounts/o8/id',
      repo_name: 'usertestrepo',
      vm_uuid: @vm_uuid,
      user: {
        first_name: 'test_first_name',
        email: 'foo@example.com'
      }
    }

    assert_response :success
    response_items = JSON.parse(@response.body)['items']
    response_object = find_obj_in_resp response_items, 'User', nil
    assert_not_nil response_object['uuid'], 'expected uuid for new user'
    assert_equal response_object['email'], 'foo@example.com', 'expected given email'
    assert_equal 'test_first_name', response_object['first_name'],
        'expecting first name'

    # five extra links; system_group, login, group, repo and vm
    verify_num_links @all_links_at_start, 5
  end

  test "setup user with an existing user email and check different object is created" do
    authorize_with :admin
    inactive_user = users(:inactive)

    post :setup, {
      openid_prefix: 'https://www.google.com/accounts/o8/id',
      repo_name: 'usertestrepo',
      user: {
        email: inactive_user['email']
      }
    }

    assert_response :success
    response_items = JSON.parse(@response.body)['items']
    response_object = find_obj_in_resp response_items, 'User', nil
    assert_not_nil response_object['uuid'], 'expected uuid for new user'
    assert_not_equal response_object['uuid'], inactive_user['uuid'],
        'expected different uuid after create operation'
    assert_equal inactive_user['email'], response_object['email'], 'expected given email'
    # system_group, openid, group, and repo. No vm link.
    verify_num_links @all_links_at_start, 4
  end

  test "setup user with openid prefix" do
    authorize_with :admin

    post :setup, {
      repo_name: 'usertestrepo',
      openid_prefix: 'http://www.example.com/account',
      user: {
        first_name: "in_create_test_first_name",
        last_name: "test_last_name",
        email: "foo@example.com"
      }
    }

    assert_response :success

    response_items = JSON.parse(@response.body)['items']
    created = find_obj_in_resp response_items, 'User', nil

    assert_equal 'in_create_test_first_name', created['first_name']
    assert_not_nil created['uuid'], 'expected uuid for new user'
    assert_not_nil created['email'], 'expected non-nil email'
    assert_nil created['identity_url'], 'expected no identity_url'

    # verify links
    # four new links: system_group, arvados#user, repo, and 'All users' group.
    verify_num_links @all_links_at_start, 4

    verify_link response_items, 'arvados#user', true, 'permission', 'can_login',
        created['uuid'], created['email'], 'arvados#user', false, 'User'

    verify_link response_items, 'arvados#repository', true, 'permission', 'can_manage',
        'foo/usertestrepo', created['uuid'], 'arvados#repository', true, 'Repository'

    verify_link response_items, 'arvados#group', true, 'permission', 'can_read',
        'All users', created['uuid'], 'arvados#group', true, 'Group'

    verify_link response_items, 'arvados#virtualMachine', false, 'permission', 'can_login',
        nil, created['uuid'], 'arvados#virtualMachine', false, 'VirtualMachine'
  end

  test "invoke setup with no openid prefix, expect error" do
    authorize_with :admin

    post :setup, {
      repo_name: 'usertestrepo',
      user: {
        first_name: "in_create_test_first_name",
        last_name: "test_last_name",
        email: "foo@example.com"
      }
    }

    response_body = JSON.parse(@response.body)
    response_errors = response_body['errors']
    assert_not_nil response_errors, 'Expected error in response'
    assert (response_errors.first.include? 'openid_prefix parameter is missing'),
        'Expected ArgumentError'
  end

  test "setup user with user, vm and repo and verify links" do
    authorize_with :admin

    post :setup, {
      user: {
        first_name: "in_create_test_first_name",
        last_name: "test_last_name",
        email: "foo@example.com"
      },
      vm_uuid: @vm_uuid,
      repo_name: 'usertestrepo',
      openid_prefix: 'https://www.google.com/accounts/o8/id'
    }

    assert_response :success

    response_items = JSON.parse(@response.body)['items']
    created = find_obj_in_resp response_items, 'User', nil

    assert_equal 'in_create_test_first_name', created['first_name']
    assert_not_nil created['uuid'], 'expected uuid for new user'
    assert_not_nil created['email'], 'expected non-nil email'
    assert_nil created['identity_url'], 'expected no identity_url'

    # five new links: system_group, arvados#user, repo, vm and 'All
    # users' group link
    verify_num_links @all_links_at_start, 5

    verify_link response_items, 'arvados#user', true, 'permission', 'can_login',
        created['uuid'], created['email'], 'arvados#user', false, 'User'

    verify_link response_items, 'arvados#repository', true, 'permission', 'can_manage',
        'foo/usertestrepo', created['uuid'], 'arvados#repository', true, 'Repository'

    verify_link response_items, 'arvados#group', true, 'permission', 'can_read',
        'All users', created['uuid'], 'arvados#group', true, 'Group'

    verify_link response_items, 'arvados#virtualMachine', true, 'permission', 'can_login',
        @vm_uuid, created['uuid'], 'arvados#virtualMachine', false, 'VirtualMachine'
  end

  test "create user as non admin user and expect error" do
    authorize_with :active

    post :create, {
      user: {email: 'foo@example.com'}
    }

    response_body = JSON.parse(@response.body)
    response_errors = response_body['errors']
    assert_not_nil response_errors, 'Expected error in response'
    assert (response_errors.first.include? 'PermissionDenied'),
          'Expected PermissionDeniedError'
  end

  test "setup user as non admin user and expect error" do
    authorize_with :active

    post :setup, {
      openid_prefix: 'https://www.google.com/accounts/o8/id',
      user: {email: 'foo@example.com'}
    }

    response_body = JSON.parse(@response.body)
    response_errors = response_body['errors']
    assert_not_nil response_errors, 'Expected error in response'
    assert (response_errors.first.include? 'Forbidden'),
          'Expected Forbidden error'
  end

  test "setup active user with repo and no vm" do
    authorize_with :admin
    active_user = users(:active)

    # invoke setup with a repository
    post :setup, {
      repo_name: 'usertestrepo',
      uuid: active_user['uuid']
    }

    assert_response :success

    response_items = JSON.parse(@response.body)['items']
    created = find_obj_in_resp response_items, 'User', nil

    assert_equal active_user[:email], created['email'], 'expected input email'

     # verify links
    verify_link response_items, 'arvados#group', true, 'permission', 'can_read',
        'All users', created['uuid'], 'arvados#group', true, 'Group'

    verify_link response_items, 'arvados#repository', true, 'permission', 'can_manage',
        'active/usertestrepo', created['uuid'], 'arvados#repository', true, 'Repository'

    verify_link response_items, 'arvados#virtualMachine', false, 'permission', 'can_login',
        nil, created['uuid'], 'arvados#virtualMachine', false, 'VirtualMachine'
  end

  test "setup active user with vm and no repo" do
    authorize_with :admin
    active_user = users(:active)
    repos_query = Repository.where(owner_uuid: active_user.uuid)
    repo_link_query = Link.where(tail_uuid: active_user.uuid,
                                 link_class: "permission", name: "can_manage")
    repos_count = repos_query.count
    repo_link_count = repo_link_query.count

    # invoke setup with a repository
    post :setup, {
      vm_uuid: @vm_uuid,
      uuid: active_user['uuid'],
      email: 'junk_email'
    }

    assert_response :success

    response_items = JSON.parse(@response.body)['items']
    created = find_obj_in_resp response_items, 'User', nil

    assert_equal active_user['email'], created['email'], 'expected original email'

    # verify links
    verify_link response_items, 'arvados#group', true, 'permission', 'can_read',
        'All users', created['uuid'], 'arvados#group', true, 'Group'

    assert_equal(repos_count, repos_query.count)
    assert_equal(repo_link_count, repo_link_query.count)

    verify_link response_items, 'arvados#virtualMachine', true, 'permission', 'can_login',
        @vm_uuid, created['uuid'], 'arvados#virtualMachine', false, 'VirtualMachine'
  end

  test "unsetup active user" do
    active_user = users(:active)
    assert_not_nil active_user['uuid'], 'expected uuid for the active user'
    assert active_user['is_active'], 'expected is_active for active user'

    verify_link_existence active_user['uuid'], active_user['email'],
          false, true, true, true, true

    authorize_with :admin

    # now unsetup this user
    post :unsetup, id: active_user['uuid']
    assert_response :success

    response_user = JSON.parse(@response.body)
    assert_not_nil response_user['uuid'], 'expected uuid for the upsetup user'
    assert_equal active_user['uuid'], response_user['uuid'], 'expected uuid not found'
    assert !response_user['is_active'], 'expected user to be inactive'
    assert !response_user['is_invited'], 'expected user to be uninvited'

    verify_link_existence response_user['uuid'], response_user['email'],
          false, false, false, false, false

    active_user = User.find_by_uuid(users(:active).uuid)
    readable_groups = active_user.groups_i_can(:read)
    all_users_group = Group.all.collect(&:uuid).select { |g| g.match /-f+$/ }
    refute_includes(readable_groups, all_users_group,
                    "active user can read All Users group after being deactivated")
    assert_equal(false, active_user.is_invited,
                 "active user is_invited after being deactivated & reloaded")
  end

  test "setup user with send notification param false and verify no email" do
    authorize_with :admin

    post :setup, {
      openid_prefix: 'http://www.example.com/account',
      send_notification_email: 'false',
      user: {
        email: "foo@example.com"
      }
    }

    assert_response :success
    response_items = JSON.parse(@response.body)['items']
    created = find_obj_in_resp response_items, 'User', nil
    assert_not_nil created['uuid'], 'expected uuid for the new user'
    assert_equal created['email'], 'foo@example.com', 'expected given email'

    setup_email = ActionMailer::Base.deliveries.last
    assert_nil setup_email, 'expected no setup email'
  end

  test "setup user with send notification param true and verify email" do
    authorize_with :admin

    post :setup, {
      openid_prefix: 'http://www.example.com/account',
      send_notification_email: 'true',
      user: {
        email: "foo@example.com"
      }
    }

    assert_response :success
    response_items = JSON.parse(@response.body)['items']
    created = find_obj_in_resp response_items, 'User', nil
    assert_not_nil created['uuid'], 'expected uuid for the new user'
    assert_equal created['email'], 'foo@example.com', 'expected given email'

    setup_email = ActionMailer::Base.deliveries.last
    assert_not_nil setup_email, 'Expected email after setup'

    assert_equal Rails.configuration.user_notifier_email_from, setup_email.from[0]
    assert_equal 'foo@example.com', setup_email.to[0]
    assert_equal 'Welcome to Curoverse', setup_email.subject
    assert (setup_email.body.to_s.include? 'Your Arvados account has been set up'),
        'Expected Your Arvados account has been set up in email body'
    assert (setup_email.body.to_s.include? 'foo@example.com'),
        'Expected user email in email body'
    assert (setup_email.body.to_s.include? Rails.configuration.workbench_address),
        'Expected workbench url in email body'
  end

  test "non-admin user can get basic information about readable users" do
    authorize_with :spectator
    get(:index)
    check_non_admin_index
    check_readable_users_index [:spectator], [:inactive, :active]
  end

  test "non-admin user gets only safe attributes from users#show" do
    g = act_as_system_user do
      create :group
    end
    users = create_list :active_user, 2, join_groups: [g]
    token = create :token, user: users[0]
    authorize_with_token token
    get :show, id: users[1].uuid
    check_non_admin_show
  end

  [2, 4].each do |limit|
    test "non-admin user can limit index to #{limit}" do
      g = act_as_system_user do
        create :group
      end
      users = create_list :active_user, 4, join_groups: [g]
      token = create :token, user: users[0]

      authorize_with_token token
      get(:index, limit: limit)
      check_non_admin_index
      assert_equal(limit, json_response["items"].size,
                   "non-admin index limit was ineffective")
    end
  end

  test "admin has full index powers" do
    authorize_with :admin
    check_inactive_user_findable
  end

  test "reader token can grant admin index powers" do
    authorize_with :spectator
    check_inactive_user_findable(reader_tokens: [api_token(:admin)])
  end

  test "admin can filter on user.is_active" do
    authorize_with :admin
    get(:index, filters: [["is_active", "=", "true"]])
    assert_response :success
    check_readable_users_index [:active, :spectator], [:inactive]
  end

  test "admin can search where user.is_active" do
    authorize_with :admin
    get(:index, where: {is_active: true})
    assert_response :success
    check_readable_users_index [:active, :spectator], [:inactive]
  end

  test "update active_no_prefs user profile and expect notification email" do
    authorize_with :admin

    put :update, {
      id: users(:active_no_prefs).uuid,
      user: {
        prefs: {:profile => {'organization' => 'example.com'}}
      }
    }
    assert_response :success

    found_email = false
    ActionMailer::Base.deliveries.andand.each do |email|
      if email.subject == "Profile created by #{users(:active_no_prefs).email}"
        found_email = true
        break
      end
    end
    assert_equal true, found_email, 'Expected email after creating profile'
  end

  test "update active_no_prefs_profile user profile and expect notification email" do
    authorize_with :admin

    user = {}
    user[:prefs] = users(:active_no_prefs_profile_no_getting_started_shown).prefs
    user[:prefs][:profile] = {:profile => {'organization' => 'example.com'}}
    put :update, {
      id: users(:active_no_prefs_profile_no_getting_started_shown).uuid,
      user: user
    }
    assert_response :success

    found_email = false
    ActionMailer::Base.deliveries.andand.each do |email|
      if email.subject == "Profile created by #{users(:active_no_prefs_profile_no_getting_started_shown).email}"
        found_email = true
        break
      end
    end
    assert_equal true, found_email, 'Expected email after creating profile'
  end

  test "update active user profile and expect no notification email" do
    authorize_with :admin

    put :update, {
      id: users(:active).uuid,
      user: {
        prefs: {:profile => {'organization' => 'anotherexample.com'}}
      }
    }
    assert_response :success

    found_email = false
    ActionMailer::Base.deliveries.andand.each do |email|
      if email.subject == "Profile created by #{users(:active).email}"
        found_email = true
        break
      end
    end
    assert_equal false, found_email, 'Expected no email after updating profile'
  end

  test "user API response includes writable_by" do
    authorize_with :active
    get :current
    assert_response :success
    assert_includes(json_response["writable_by"], users(:active).uuid,
                    "user's writable_by should include self")
    assert_includes(json_response["writable_by"], users(:active).owner_uuid,
                    "user's writable_by should include its owner_uuid")
  end


  NON_ADMIN_USER_DATA = ["uuid", "kind", "is_active", "email", "first_name",
                         "last_name"].sort

  def check_non_admin_index
    assert_response :success
    response_items = json_response["items"]
    assert_not_nil response_items
    response_items.each do |user_data|
      check_non_admin_item user_data
      assert(user_data["is_active"], "non-admin index returned inactive user")
    end
  end

  def check_non_admin_show
    assert_response :success
    check_non_admin_item json_response
  end

  def check_non_admin_item user_data
    assert_equal(NON_ADMIN_USER_DATA, user_data.keys.sort,
                 "data in response had missing or extra attributes")
    assert_equal("arvados#user", user_data["kind"])
  end


  def check_readable_users_index expect_present, expect_missing
    response_uuids = json_response["items"].map { |u| u["uuid"] }
    expect_present.each do |user_key|
      assert_includes(response_uuids, users(user_key).uuid,
                      "#{user_key} missing from index")
    end
    expect_missing.each do |user_key|
      refute_includes(response_uuids, users(user_key).uuid,
                      "#{user_key} included in index")
    end
  end

  def check_inactive_user_findable(params={})
    inactive_user = users(:inactive)
    get(:index, params.merge(filters: [["email", "=", inactive_user.email]]))
    assert_response :success
    user_list = json_response["items"]
    assert_equal(1, user_list.andand.count)
    # This test needs to check a column non-admins have no access to,
    # to ensure that admins see all user information.
    assert_equal(inactive_user.identity_url, user_list.first["identity_url"],
                 "admin's filtered index did not return inactive user")
  end

  def verify_num_links (original_links, expected_additional_links)
    links_now = Link.all
    assert_equal expected_additional_links, Link.all.size-original_links.size,
        "Expected #{expected_additional_links.inspect} more links"
  end

  def find_obj_in_resp (response_items, object_type, head_kind=nil)
    return_obj = nil
    response_items
    response_items.each { |x|
      if !x
        next
      end

      if object_type == 'User'
        if ArvadosModel::resource_class_for_uuid(x['uuid']) == User
          return_obj = x
          break
        end
      else  # looking for a link
        if x['head_uuid'] and ArvadosModel::resource_class_for_uuid(x['head_uuid']).kind == head_kind
          return_obj = x
          break
        end
      end
    }
    return return_obj
  end
end
