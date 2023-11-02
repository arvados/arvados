# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

# The v1 API uses token scopes to control access to the REST API at the path
# level.  This is enforced in the base ApplicationController, making it a
# functional test that we can run against many different controllers.

require 'test_helper'

class ApiTokensScopeTest < ActionDispatch::IntegrationTest
  fixtures :all

  def v1_url(*parts)
    (['', 'arvados', 'v1'] + parts).join('/')
  end

  test "user list token can only list users" do
    get_args = {params: {}, headers: auth(:active_userlist)}
    get(v1_url('users'), **get_args)
    assert_response :success
    get(v1_url('users', ''), **get_args)  # Add trailing slash.
    assert_response :success
    get(v1_url('users', 'current'), **get_args)
    assert_response 403
    get(v1_url('virtual_machines'), **get_args)
    assert_response 403
  end

  test "narrow + wide scoped tokens for different users" do
    get_args = {
      params: {
        reader_tokens: [api_client_authorizations(:anonymous).api_token]
      },
      headers: auth(:active_userlist),
    }
    get(v1_url('users'), **get_args)
    assert_response :success
    get(v1_url('users', ''), **get_args)  # Add trailing slash.
    assert_response :success
    get(v1_url('users', 'current'), **get_args)
    assert_response 403
    get(v1_url('virtual_machines'), **get_args)
    assert_response 403
   end

  test "specimens token can see exactly owned specimens" do
    get_args = {params: {}, headers: auth(:active_specimens)}
    get(v1_url('specimens'), **get_args)
    assert_response 403
    get(v1_url('specimens', specimens(:owned_by_active_user).uuid), **get_args)
    assert_response :success
    head(v1_url('specimens', specimens(:owned_by_active_user).uuid), **get_args)
    assert_response :success
    get(v1_url('specimens', specimens(:owned_by_spectator).uuid), **get_args)
    assert_includes(403..404, @response.status)
  end

  test "token with multiple scopes can use them all" do
    def get_token_count
      get(v1_url('api_client_authorizations'),
        params: {},
        headers: auth(:active_apitokens))
      assert_response :success
      token_count = JSON.parse(@response.body)['items_available']
      assert_not_nil(token_count, "could not find token count")
      token_count
    end
    # Test the GET scope.
    token_count = get_token_count
    # Test the POST scope.
    post(v1_url('api_client_authorizations'),
      params: {api_client_authorization: {user_id: users(:active).id}},
      headers: auth(:active_apitokens))
    assert_response :success
    assert_equal(token_count + 1, get_token_count,
                 "token count suggests POST was not accepted")
    # Test other requests are denied.
    get(v1_url('api_client_authorizations',
               api_client_authorizations(:active_apitokens).uuid),
        params: {}, headers: auth(:active_apitokens))
    assert_response 403
  end

  test "token without scope has no access" do
    # Logs are good for this test, because logs have relatively
    # few access controls enforced at the model level.
    req_args = {params: {}, headers: auth(:admin_noscope)}
    get(v1_url('logs'), **req_args)
    assert_response 403
    get(v1_url('logs', logs(:noop).uuid), **req_args)
    assert_response 403
    post(v1_url('logs'), **req_args)
    assert_response 403
  end

  test "VM login scopes work" do
    # A system administration script makes an API token with limited scope
    # for virtual machines to let it see logins.
    def vm_logins_url(name)
      v1_url('virtual_machines', virtual_machines(name).uuid, 'logins')
    end
    get_args = {params: {}, headers: auth(:admin_vm)}
    get(vm_logins_url(:testvm), **get_args)
    assert_response :success
    get(vm_logins_url(:testvm2), **get_args)
    assert_includes(400..419, @response.status,
                    "getting testvm2 logins should have failed")
  end
end
