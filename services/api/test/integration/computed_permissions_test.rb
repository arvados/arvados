# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

require 'test_helper'

class ComputedPermissionsTest < ActionDispatch::IntegrationTest
  include DbCurrentTime
  fixtures :users, :groups, :api_client_authorizations, :collections

  test "non-admin forbidden" do
    get "/arvados/v1/computed_permissions",
      params: {:format => :json},
      headers: auth(:active)
    assert_response 403
  end

  test "admin get permission for specified user" do
    get "/arvados/v1/computed_permissions",
      params: {
        :format => :json,
        :filters => [['user_uuid', '=', users(:active).uuid]].to_json,
      },
      headers: auth(:admin)
    assert_response :success
    assert_equal users(:active).uuid, json_response['items'][0]['user_uuid']
    assert_nil json_response['count']
  end

  test "admin get implicit permission for specified user and target" do
    get "/arvados/v1/computed_permissions",
      params: {
        :format => :json,
        :filters => [
          ['user_uuid', '=', users(:active).uuid],
          ['target_uuid', '=', groups(:private).uuid],
        ].to_json,
      },
      headers: auth(:admin)
    assert_response :success
    assert_equal 1, json_response['items'].length
    assert_equal users(:active).uuid, json_response['items'][0]['user_uuid']
    assert_equal groups(:private).uuid, json_response['items'][0]['target_uuid']
    assert_equal 'can_manage', json_response['items'][0]['perm_level']
  end

  test "reject count=exact" do
    get "/arvados/v1/computed_permissions",
      params: {
        :format => :json,
        :count => 'exact',
      },
      headers: auth(:admin)
    assert_response 422
  end

  test "reject offset>0" do
    get "/arvados/v1/computed_permissions",
      params: {
        :format => :json,
        :offset => 7,
      },
      headers: auth(:admin)
    assert_response 422
  end
end
