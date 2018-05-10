# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

require 'test_helper'
require 'helpers/users_test_helper'

class UsersTest < ActionDispatch::IntegrationTest
  include UsersTestHelper

  test "setup user multiple times" do
    repo_name = 'usertestrepo'

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
        'foo/usertestrepo', created['uuid'], 'arvados#repository', true, 'Repository'

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
    assert_response 422         # cannot create another user with same UUID

    # invoke setup on the same user
    post "/arvados/v1/users/setup", {
      repo_name: repo_name,
      vm_uuid: virtual_machines(:testvm).uuid,
      openid_prefix: 'https://www.google.com/accounts/o8/id',
      uuid: 'zzzzz-tpzed-abcdefghijklmno',
    }, auth(:admin)

    response_items = json_response['items']

    created = find_obj_in_resp response_items, 'arvados#user', nil
    assert_equal 'in_create_test_first_name', created['first_name']
    assert_not_nil created['uuid'], 'expected non-null uuid for the new user'
    assert_equal 'zzzzz-tpzed-abcdefghijklmno', created['uuid']
    assert_not_nil created['email'], 'expected non-nil email'
    assert_nil created['identity_url'], 'expected no identity_url'

    # arvados#user, repo link and link add user to 'All users' group
    verify_link response_items, 'arvados#repository', true, 'permission', 'can_manage',
        'foo/usertestrepo', created['uuid'], 'arvados#repository', true, 'Repository'

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

    verify_link response_items, 'arvados#virtualMachine', false, 'permission', 'can_login',
        nil, created['uuid'], 'arvados#virtualMachine', false, 'VirtualMachine'

   # invoke setup with a repository
    post "/arvados/v1/users/setup", {
      openid_prefix: 'http://www.example.com/account',
      repo_name: 'newusertestrepo',
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
        'foo/newusertestrepo', created['uuid'], 'arvados#repository', true, 'Repository'

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

    verify_link response_items, 'arvados#virtualMachine', true, 'permission', 'can_login',
        virtual_machines(:testvm).uuid, created['uuid'], 'arvados#virtualMachine', false, 'VirtualMachine'
  end

  test "setup and unsetup user" do
    post "/arvados/v1/users/setup", {
      repo_name: 'newusertestrepo',
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
        'foo/newusertestrepo', created['uuid'], 'arvados#repository', true, 'Repository'

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

  test 'merge active into project_viewer account' do
    post('/arvados/v1/groups', {
           group: {
             group_class: 'project',
             name: "active user's stuff",
           },
         }, auth(:project_viewer))
    assert_response(:success)
    project_uuid = json_response['uuid']

    post('/arvados/v1/users/merge', {
           new_user_token: api_client_authorizations(:project_viewer_trustedclient).api_token,
           new_owner_uuid: project_uuid,
           redirect_to_new_user: true,
         }, auth(:active_trustedclient))
    assert_response(:success)

    get('/arvados/v1/users/current', {}, auth(:active))
    assert_response(:success)
    assert_equal(users(:project_viewer).uuid, json_response['uuid'])

    get('/arvados/v1/authorized_keys/' + authorized_keys(:active).uuid, {}, auth(:active))
    assert_response(:success)
    assert_equal(users(:project_viewer).uuid, json_response['owner_uuid'])
    assert_equal(users(:project_viewer).uuid, json_response['authorized_user_uuid'])

    get('/arvados/v1/repositories/' + repositories(:foo).uuid, {}, auth(:active))
    assert_response(:success)
    assert_equal(users(:project_viewer).uuid, json_response['owner_uuid'])

    get('/arvados/v1/groups/' + groups(:aproject).uuid, {}, auth(:active))
    assert_response(:success)
    assert_equal(project_uuid, json_response['owner_uuid'])
  end
end
