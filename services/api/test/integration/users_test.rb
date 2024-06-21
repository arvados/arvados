# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

require 'test_helper'
require 'helpers/users_test_helper'

class UsersTest < ActionDispatch::IntegrationTest
  include UsersTestHelper

  test "setup user multiple times" do
    post "/arvados/v1/users/setup",
      params: {
        user: {
          uuid: 'zzzzz-tpzed-abcdefghijklmno',
          first_name: "in_create_test_first_name",
          last_name: "test_last_name",
          email: "foo@example.com"
        }
      },
      headers: auth(:admin)

    assert_response :success

    response_items = json_response['items']

    created = find_obj_in_resp response_items, 'arvados#user', nil

    assert_equal 'in_create_test_first_name', created['first_name']
    assert_not_nil created['uuid'], 'expected non-null uuid for the new user'
    assert_equal 'zzzzz-tpzed-abcdefghijklmno', created['uuid']
    assert_not_nil created['email'], 'expected non-nil email'
    assert_nil created['identity_url'], 'expected no identity_url'

    # link to add user to 'All users' group

    verify_link response_items, 'arvados#group', true, 'permission', 'can_write',
        'All users', created['uuid'], 'arvados#group', true, 'Group'

    verify_link response_items, 'arvados#virtualMachine', false, 'permission', 'can_login',
        nil, created['uuid'], 'arvados#virtualMachine', false, 'VirtualMachine'

    verify_system_group_permission_link_for created['uuid']

    # invoke setup again with the same data
    post "/arvados/v1/users/setup",
      params: {
        vm_uuid: virtual_machines(:testvm).uuid,
        user: {
          uuid: 'zzzzz-tpzed-abcdefghijklmno',
          first_name: "in_create_test_first_name",
          last_name: "test_last_name",
          email: "foo@example.com"
        }
      },
      headers: auth(:admin)
    assert_response 422         # cannot create another user with same UUID

    # invoke setup on the same user
    post "/arvados/v1/users/setup",
      params: {
        vm_uuid: virtual_machines(:testvm).uuid,
        uuid: 'zzzzz-tpzed-abcdefghijklmno',
      },
      headers: auth(:admin)

    response_items = json_response['items']

    created = find_obj_in_resp response_items, 'arvados#user', nil
    assert_equal 'in_create_test_first_name', created['first_name']
    assert_not_nil created['uuid'], 'expected non-null uuid for the new user'
    assert_equal 'zzzzz-tpzed-abcdefghijklmno', created['uuid']
    assert_not_nil created['email'], 'expected non-nil email'
    assert_nil created['identity_url'], 'expected no identity_url'

    # arvados#user, and link to add user to 'All users' group
    verify_link response_items, 'arvados#group', true, 'permission', 'can_write',
        'All users', created['uuid'], 'arvados#group', true, 'Group'

    verify_link response_items, 'arvados#virtualMachine', true, 'permission', 'can_login',
        virtual_machines(:testvm).uuid, created['uuid'], 'arvados#virtualMachine', false, 'VirtualMachine'

    verify_system_group_permission_link_for created['uuid']
  end

  test "setup user in multiple steps and verify response" do
    post "/arvados/v1/users/setup",
      params: {
        user: {
          email: "foo@example.com"
        }
      },
      headers: auth(:admin)

    assert_response :success
    response_items = json_response['items']
    created = find_obj_in_resp response_items, 'arvados#user', nil

    assert_not_nil created['uuid'], 'expected uuid for new user'
    assert_not_nil created['email'], 'expected non-nil email'
    assert_equal created['email'], 'foo@example.com', 'expected input email'

    # two new links: system_group, and 'All users' group.

    verify_link response_items, 'arvados#group', true, 'permission', 'can_write',
        'All users', created['uuid'], 'arvados#group', true, 'Group'

    verify_link response_items, 'arvados#virtualMachine', false, 'permission', 'can_login',
        nil, created['uuid'], 'arvados#virtualMachine', false, 'VirtualMachine'

    # invoke setup with a vm_uuid
    post "/arvados/v1/users/setup",
      params: {
        vm_uuid: virtual_machines(:testvm).uuid,
        user: {
          email: 'junk_email'
        },
        uuid: created['uuid']
      },
      headers: auth(:admin)

    assert_response :success

    response_items = json_response['items']
    created = find_obj_in_resp response_items, 'arvados#user', nil

    assert_equal created['email'], 'foo@example.com', 'expected original email'

    # verify links
    verify_link response_items, 'arvados#group', true, 'permission', 'can_write',
        'All users', created['uuid'], 'arvados#group', true, 'Group'

    verify_link response_items, 'arvados#virtualMachine', true, 'permission', 'can_login',
        virtual_machines(:testvm).uuid, created['uuid'], 'arvados#virtualMachine', false, 'VirtualMachine'
  end

  test "setup and unsetup user" do
    post "/arvados/v1/users/setup",
      params: {
        vm_uuid: virtual_machines(:testvm).uuid,
        user: {email: 'foo@example.com'},
      },
      headers: auth(:admin)

    assert_response :success
    response_items = json_response['items']
    created = find_obj_in_resp response_items, 'arvados#user', nil
    assert_not_nil created['uuid'], 'expected uuid for the new user'
    assert_equal created['email'], 'foo@example.com', 'expected given email'

    # three extra links: system_group, login, group and vm

    verify_link response_items, 'arvados#group', true, 'permission', 'can_write',
        'All users', created['uuid'], 'arvados#group', true, 'Group'

    verify_link response_items, 'arvados#virtualMachine', true, 'permission', 'can_login',
        virtual_machines(:testvm).uuid, created['uuid'], 'arvados#virtualMachine', false, 'VirtualMachine'

    verify_link_existence created['uuid'], created['email'], true, true, true, true, false

    # create a token
    token = act_as_system_user do
      ApiClientAuthorization.create!(user: User.find_by_uuid(created['uuid']), api_client: ApiClient.all.first).api_token
    end

    # share project and collections with the new user
    act_as_system_user do
      Link.create!(tail_uuid: created['uuid'],
                   head_uuid: groups(:aproject).uuid,
                   link_class: 'permission',
                   name: 'can_manage')
      Link.create!(tail_uuid: created['uuid'],
                   head_uuid: collections(:collection_owned_by_active).uuid,
                   link_class: 'permission',
                   name: 'can_read')
      Link.create!(tail_uuid: created['uuid'],
                   head_uuid: collections(:collection_owned_by_active_with_file_stats).uuid,
                   link_class: 'permission',
                   name: 'can_write')
    end

    assert_equal 1, ApiClientAuthorization.where(user_id: User.find_by_uuid(created['uuid']).id).size, 'expected token not found'

    post "/arvados/v1/users/#{created['uuid']}/unsetup", params: {}, headers: auth(:admin)

    assert_response :success

    created2 = json_response
    assert_not_nil created2['uuid'], 'expected uuid for the newly created user'
    assert_equal created['uuid'], created2['uuid'], 'expected uuid not found'
    assert_equal 0, ApiClientAuthorization.where(user_id: User.find_by_uuid(created['uuid']).id).size, 'token should have been deleted by user unsetup'
    # check permissions are deleted
    assert_empty Link.where(tail_uuid: created['uuid'])

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
    post('/arvados/v1/groups',
      params: {
        group: {
          group_class: 'project',
          name: "active user's stuff",
        },
      },
      headers: auth(:project_viewer))
    assert_response(:success)
    project_uuid = json_response['uuid']

    post('/arvados/v1/users/merge',
      params: {
        new_user_token: api_client_authorizations(:project_viewer_trustedclient).api_token,
        new_owner_uuid: project_uuid,
        redirect_to_new_user: true,
      },
      headers: auth(:active_trustedclient))
    assert_response(:success)

    get('/arvados/v1/users/current', params: {}, headers: auth(:active))
    assert_response(:success)
    assert_equal(users(:project_viewer).uuid, json_response['uuid'])

    get('/arvados/v1/authorized_keys/' + authorized_keys(:active).uuid,
      params: {},
      headers: auth(:active))
    assert_response(:success)
    assert_equal(users(:project_viewer).uuid, json_response['owner_uuid'])
    assert_equal(users(:project_viewer).uuid, json_response['authorized_user_uuid'])

    get('/arvados/v1/groups/' + groups(:aproject).uuid,
      params: {},
      headers: auth(:active))
    assert_response(:success)
    assert_equal(project_uuid, json_response['owner_uuid'])
  end

  test 'pre-activate user' do
    post '/arvados/v1/users',
      params: {
        "user" => {
          "email" => 'foo@example.com',
          "is_active" => true,
          "username" => "barney"
        }
      },
      headers: {'HTTP_AUTHORIZATION' => "OAuth2 #{api_token(:admin)}"}
    assert_response :success
    rp = json_response
    assert_not_nil rp["uuid"]
    assert_equal true, rp["is_active"]
    assert_equal false, rp["is_admin"]

    get "/arvados/v1/users/#{rp['uuid']}",
      params: {format: 'json'},
      headers: auth(:admin)
    assert_response :success
    assert_equal rp["uuid"], json_response['uuid']
    assert_equal false, json_response['is_admin']
    assert_equal true, json_response['is_active']
    assert_equal 'foo@example.com', json_response['email']
    assert_equal 'barney', json_response['username']
  end

  test "cannot set is_active to false directly" do
    post('/arvados/v1/users',
      params: {
        user: {
          email: "bob@example.com",
          username: "bobby"
        },
      },
      headers: auth(:admin))
    assert_response(:success)
    user = json_response
    assert_equal false, user['is_active']

    token = act_as_system_user do
      ApiClientAuthorization.create!(user: User.find_by_uuid(user['uuid']), api_client: ApiClient.all.first).api_token
    end
    post("/arvados/v1/user_agreements/sign",
        params: {uuid: 'zzzzz-4zz18-t68oksiu9m80s4y'},
        headers: {"HTTP_AUTHORIZATION" => "Bearer #{token}"})
    assert_response :success

    post("/arvados/v1/users/#{user['uuid']}/activate",
      params: {},
      headers: auth(:admin))
    assert_response(:success)
    user = json_response
    assert_equal true, user['is_active']

    put("/arvados/v1/users/#{user['uuid']}",
         params: {
           user: {is_active: false}
         },
         headers: auth(:admin))
    assert_response 422
  end

  test "cannot self activate when AutoSetupNewUsers is false" do
    Rails.configuration.Users.NewUsersAreActive = false
    Rails.configuration.Users.AutoSetupNewUsers = false

    user = nil
    token = nil
    act_as_system_user do
      user = User.create!(email: "bob@example.com", username: "bobby")
      ap = ApiClientAuthorization.create!(user: user, api_client: ApiClient.all.first)
      token = ap.api_token
    end

    get("/arvados/v1/users/#{user['uuid']}",
        params: {},
        headers: {"HTTP_AUTHORIZATION" => "Bearer #{token}"})
    assert_response(:success)
    user = json_response
    assert_equal false, user['is_active']

    post("/arvados/v1/users/#{user['uuid']}/activate",
        params: {},
        headers: {"HTTP_AUTHORIZATION" => "Bearer #{token}"})
    assert_response 422
    assert_match(/Cannot activate without being invited/, json_response['errors'][0])
  end


  test "cannot self activate after unsetup" do
    Rails.configuration.Users.NewUsersAreActive = false
    Rails.configuration.Users.AutoSetupNewUsers = false

    user = nil
    token = nil
    act_as_system_user do
      user = User.create!(email: "bob@example.com", username: "bobby")
      ap = ApiClientAuthorization.create!(user: user, api_client_id: 0)
      token = ap.api_token
    end

    post("/arvados/v1/users/setup",
        params: {uuid: user['uuid']},
        headers: auth(:admin))
    assert_response :success

    post("/arvados/v1/users/#{user['uuid']}/activate",
        params: {},
        headers: {"HTTP_AUTHORIZATION" => "Bearer #{token}"})
    assert_response 403
    assert_match(/Cannot activate without user agreements/, json_response['errors'][0])

    post("/arvados/v1/user_agreements/sign",
        params: {uuid: 'zzzzz-4zz18-t68oksiu9m80s4y'},
        headers: {"HTTP_AUTHORIZATION" => "Bearer #{token}"})
    assert_response :success

    post("/arvados/v1/users/#{user['uuid']}/activate",
        params: {},
        headers: {"HTTP_AUTHORIZATION" => "Bearer #{token}"})
    assert_response :success

    get("/arvados/v1/users/#{user['uuid']}",
        params: {},
        headers: {"HTTP_AUTHORIZATION" => "Bearer #{token}"})
    assert_response(:success)
    userJSON = json_response
    assert_equal true, userJSON['is_active']

    post("/arvados/v1/users/#{user['uuid']}/unsetup",
        params: {},
        headers: auth(:admin))
    assert_response :success

    # Need to get a new token, the old one was invalidated by the unsetup call
    act_as_system_user do
      ap = ApiClientAuthorization.create!(user: user, api_client_id: 0)
      token = ap.api_token
    end

    get("/arvados/v1/users/#{user['uuid']}",
        params: {},
        headers: {"HTTP_AUTHORIZATION" => "Bearer #{token}"})
    assert_response(:success)
    userJSON = json_response
    assert_equal false, userJSON['is_active']

    post("/arvados/v1/users/#{user['uuid']}/activate",
        params: {},
        headers: {"HTTP_AUTHORIZATION" => "Bearer #{token}"})
    assert_response 422
    assert_match(/Cannot activate without being invited/, json_response['errors'][0])
  end

  test "bypass_federation only accepted for admins" do
    get "/arvados/v1/users",
      params: {
        bypass_federation: true
      },
      headers: auth(:admin)

    assert_response :success

    get "/arvados/v1/users",
      params: {
        bypass_federation: true
      },
      headers: auth(:active)

    assert_response 403
  end

  test "disabling system root user not permitted" do
    put("/arvados/v1/users/#{users(:system_user).uuid}",
      params: {
        user: {is_admin: false}
      },
      headers: auth(:admin))
    assert_response 422

    post("/arvados/v1/users/#{users(:system_user).uuid}/unsetup",
      params: {},
      headers: auth(:admin))
    assert_response 422
  end

  test "creating users only accepted for admins" do
    assert_equal false, users(:active).is_admin
    post '/arvados/v1/users',
      params: {
        "user" => {
          "email" => 'foo@example.com',
          "username" => "barney"
        }
      },
      headers: auth(:active)
    assert_response 403
  end

  test "create users assigns the system root user as their owner" do
    post '/arvados/v1/users',
      params: {
        "user" => {
          "email" => 'foo@example.com',
          "username" => "barney"
        }
      },
      headers: auth(:admin)
    assert_response :success
    assert_not_nil json_response["uuid"]
    assert_equal users(:system_user).uuid, json_response["owner_uuid"]
  end

  test "create users ignores provided owner_uuid field" do
    assert_equal false, users(:admin).uuid == users(:system_user).uuid
    post '/arvados/v1/users',
      params: {
        "user" => {
          "email" => 'foo@example.com',
          "owner_uuid" => users(:admin).uuid,
          "username" => "barney"
        }
      },
      headers: auth(:admin)
    assert_response :success
    assert_not_nil json_response["uuid"]
    assert_equal users(:system_user).uuid, json_response["owner_uuid"]
  end
end
