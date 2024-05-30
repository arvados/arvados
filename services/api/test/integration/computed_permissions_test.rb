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
  end
end
