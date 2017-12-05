# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

require 'test_helper'

class UserTest < ActiveSupport::TestCase
  test "can select specific user columns" do
    use_token :admin
    User.select(["uuid", "is_active"]).limit(5).each do |user|
      assert_not_nil user.uuid
      assert_not_nil user.is_active
      assert_nil user.first_name
    end
  end

  test "User.current doesn't return anonymous user when using invalid token" do
    # Set up anonymous user token
    Rails.configuration.anonymous_user_token = api_fixture('api_client_authorizations')['anonymous']['api_token']
    # First, try with a valid user
    use_token :active
    u = User.current
    assert(find_fixture(User, "active").uuid == u.uuid)
    # Next, simulate an invalid token
    Thread.current[:arvados_api_token] = 'thistokenwontwork'
    assert_raises(ArvadosApiClient::NotLoggedInException) do
      User.current
    end
  end
end
