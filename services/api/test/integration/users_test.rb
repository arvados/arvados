require 'test_helper'
require 'helpers/users_test_helper'

class UsersTest < ActionDispatch::IntegrationTest
  include UsersTestHelper

  test "setup user multiple times" do
    repo_name = 'test_repo'

    post "/arvados/v1/users/setup", {
      repo_name: repo_name,
      openid_prefix: 'https://www.google.com/accounts/o8/id',
      user: {
        uuid: 'zzzzz-tpzed-abcdefghijklmno',
        first_name: "in_create_test_first_name",
        last_name: "test_last_name",
        email: "foo@example.com"
      }
    }, auth(:admin)

    assert_response :success

    response_items = json_response['items']

    created = find_obj_in_resp response_items, 'arvados#user', nil

    assert_equal 'in_create_test_first_name', created['first_name']
    assert_not_nil created['uuid'], 'expected non-null uuid for the new user'
    assert_equal 'zzzzz-tpzed-abcdefghijklmno', created['uuid']
    assert_not_nil created['email'], 'expected non-nil email'
    assert_nil created['identity_url'], 'expected no identity_url'

    # arvados#user, repo link and link add user to 'All users' group
    verify_link response_items, 'arvados#user', true, 'permission', 'can_login',
        created['uuid'], created['email'], 'arvados#user', false, 'arvados#user'

    verify_link response_items, 'arvados#repository', true, 'permission', 'can_manage',
        repo_name, created['uuid'], 'arvados#repository', true, 'Repository'

    verify_link response_items, 'arvados#group', true, 'permission', 'can_read',
        'All users', created['uuid'], 'arvados#group', true, 'Group'

    verify_link response_items, 'arvados#virtualMachine', false, 'permission', 'can_login',
        nil, created['uuid'], 'arvados#virtualMachine', false, 'VirtualMachine'

    verify_system_group_permission_link_for created['uuid']

    # invoke setup again with the same data
    post "/arvados/v1/users/setup", {
      repo_name: repo_name,
      vm_uuid: virtual_machines(:testvm).uuid,
      openid_prefix: 'https://www.google.com/accounts/o8/id',
      user: {
        uuid: 'zzzzz-tpzed-abcdefghijklmno',
        first_name: "in_create_test_first_name",
        last_name: "test_last_name",
        email: "foo@example.com"
      }
    }, auth(:admin)

    assert_response :success

    response_items = json_response['items']

    created = find_obj_in_resp response_items, 'arvados#user', nil
    assert_equal 'in_create_test_first_name', created['first_name']
    assert_not_nil created['uuid'], 'expected non-null uuid for the new user'
    assert_equal 'zzzzz-tpzed-abcdefghijklmno', created['uuid']
    assert_not_nil created['email'], 'expected non-nil email'
    assert_nil created['identity_url'], 'expected no identity_url'

    # arvados#user, repo link and link add user to 'All users' group
    verify_link response_items, 'arvados#repository', true, 'permission', 'can_manage',
        repo_name, created['uuid'], 'arvados#repository', true, 'Repository'

    verify_link response_items, 'arvados#group', true, 'permission', 'can_read',
        'All users', created['uuid'], 'arvados#group', true, 'Group'

    verify_link response_items, 'arvados#virtualMachine', true, 'permission', 'can_login',
        virtual_machines(:testvm).uuid, created['uuid'], 'arvados#virtualMachine', false, 'VirtualMachine'

    verify_system_group_permission_link_for created['uuid']
  end

  test "setup user in multiple steps and verify response" do
    post "/arvados/v1/users/setup", {
      openid_prefix: 'http://www.example.com/account',
      user: {
        email: "foo@example.com"
      }
    }, auth(:admin)

    assert_response :success
    response_items = json_response['items']
    created = find_obj_in_resp response_items, 'arvados#user', nil

    assert_not_nil created['uuid'], 'expected uuid for new user'
    assert_not_nil created['email'], 'expected non-nil email'
    assert_equal created['email'], 'foo@example.com', 'expected input email'

    # three new links: system_group, arvados#user, and 'All users' group.
    verify_link response_items, 'arvados#user', true, 'permission', 'can_login',
        created['uuid'], created['email'], 'arvados#user', false, 'arvados#user'

    verify_link response_items, 'arvados#group', true, 'permission', 'can_read',
        'All users', created['uuid'], 'arvados#group', true, 'Group'

    verify_link response_items, 'arvados#repository', false, 'permission', 'can_manage',
        'test_repo', created['uuid'], 'arvados#repository', true, 'Repository'

    verify_link response_items, 'arvados#virtualMachine', false, 'permission', 'can_login',
        nil, created['uuid'], 'arvados#virtualMachine', false, 'VirtualMachine'

   # invoke setup with a repository
    post "/arvados/v1/users/setup", {
      openid_prefix: 'http://www.example.com/account',
      repo_name: 'new_repo',
      uuid: created['uuid']
    }, auth(:admin)

    assert_response :success

    response_items = json_response['items']
    created = find_obj_in_resp response_items, 'arvados#user', nil

    assert_equal 'foo@example.com', created['email'], 'expected input email'

     # verify links
    verify_link response_items, 'arvados#group', true, 'permission', 'can_read',
        'All users', created['uuid'], 'arvados#group', true, 'Group'

    verify_link response_items, 'arvados#repository', true, 'permission', 'can_manage',
        'new_repo', created['uuid'], 'arvados#repository', true, 'Repository'

    verify_link response_items, 'arvados#virtualMachine', false, 'permission', 'can_login',
        nil, created['uuid'], 'arvados#virtualMachine', false, 'VirtualMachine'

    # invoke setup with a vm_uuid
    post "/arvados/v1/users/setup", {
      vm_uuid: virtual_machines(:testvm).uuid,
      openid_prefix: 'http://www.example.com/account',
      user: {
        email: 'junk_email'
      },
      uuid: created['uuid']
    }, auth(:admin)

    assert_response :success

    response_items = json_response['items']
    created = find_obj_in_resp response_items, 'arvados#user', nil

    assert_equal created['email'], 'foo@example.com', 'expected original email'

    # verify links
    verify_link response_items, 'arvados#group', true, 'permission', 'can_read',
        'All users', created['uuid'], 'arvados#group', true, 'Group'

    # since no repo name in input, we won't get any; even though user has one
    verify_link response_items, 'arvados#repository', false, 'permission', 'can_manage',
        'new_repo', created['uuid'], 'arvados#repository', true, 'Repository'

    verify_link response_items, 'arvados#virtualMachine', true, 'permission', 'can_login',
        virtual_machines(:testvm).uuid, created['uuid'], 'arvados#virtualMachine', false, 'VirtualMachine'
  end

  test "setup and unsetup user" do
    post "/arvados/v1/users/setup", {
      repo_name: 'test_repo',
      vm_uuid: virtual_machines(:testvm).uuid,
      user: {email: 'foo@example.com'},
      openid_prefix: 'https://www.google.com/accounts/o8/id'
    }, auth(:admin)

    assert_response :success
    response_items = json_response['items']
    created = find_obj_in_resp response_items, 'arvados#user', nil
    assert_not_nil created['uuid'], 'expected uuid for the new user'
    assert_equal created['email'], 'foo@example.com', 'expected given email'

    # five extra links: system_group, login, group, repo and vm
    verify_link response_items, 'arvados#user', true, 'permission', 'can_login',
        created['uuid'], created['email'], 'arvados#user', false, 'arvados#user'

    verify_link response_items, 'arvados#group', true, 'permission', 'can_read',
        'All users', created['uuid'], 'arvados#group', true, 'Group'

    verify_link response_items, 'arvados#repository', true, 'permission', 'can_manage',
        'test_repo', created['uuid'], 'arvados#repository', true, 'Repository'

    verify_link response_items, 'arvados#virtualMachine', true, 'permission', 'can_login',
        virtual_machines(:testvm).uuid, created['uuid'], 'arvados#virtualMachine', false, 'VirtualMachine'

    verify_link_existence created['uuid'], created['email'], true, true, true, true, false

    post "/arvados/v1/users/#{created['uuid']}/unsetup", {}, auth(:admin)

    assert_response :success

    created2 = json_response
    assert_not_nil created2['uuid'], 'expected uuid for the newly created user'
    assert_equal created['uuid'], created2['uuid'], 'expected uuid not found'

    verify_link_existence created['uuid'], created['email'], false, false, false, false, false
  end

  def find_obj_in_resp (response_items, kind, head_kind=nil)
    response_items.each do |x|
      if x && x['kind']
        return x if (x['kind'] == kind && x['head_kind'] == head_kind)
      end
    end
    nil
  end

end
