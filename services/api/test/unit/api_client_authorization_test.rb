# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

require 'test_helper'

class ApiClientAuthorizationTest < ActiveSupport::TestCase
  include CurrentApiClient

  [:admin_trustedclient, :active_trustedclient].each do |token|
    test "ApiClientAuthorization can be created then deleted by #{token}" do
      set_user_from_auth token
      x = ApiClientAuthorization.create!(user_id: current_user.id,
                                         api_client_id: 0,
                                         scopes: [])
      newtoken = x.api_token
      assert x.destroy, "Failed to destroy new ApiClientAuth"
      assert_empty ApiClientAuthorization.where(api_token: newtoken), "Destroyed ApiClientAuth is still in database"
    end
  end

  test "accepts SystemRootToken" do
    assert_nil ApiClientAuthorization.validate(token: "xxxSystemRootTokenxxx")

    # will create a new ApiClientAuthorization record
    Rails.configuration.SystemRootToken = "xxxSystemRootTokenxxx"

    auth = ApiClientAuthorization.validate(token: "xxxSystemRootTokenxxx")
    assert_equal "xxxSystemRootTokenxxx", auth.api_token
    assert_equal User.find_by_uuid(system_user_uuid).id, auth.user_id
    assert auth.api_client.is_trusted

    # now change the token and try to use the old one first
    Rails.configuration.SystemRootToken = "newxxxSystemRootTokenxxx"

    # old token will fail
    assert_nil ApiClientAuthorization.validate(token: "xxxSystemRootTokenxxx")
    # new token will work
    auth = ApiClientAuthorization.validate(token: "newxxxSystemRootTokenxxx")
    assert_equal "newxxxSystemRootTokenxxx", auth.api_token
    assert_equal User.find_by_uuid(system_user_uuid).id, auth.user_id

    # now change the token again and use the new one first
    Rails.configuration.SystemRootToken = "new2xxxSystemRootTokenxxx"

    # new token will work
    auth = ApiClientAuthorization.validate(token: "new2xxxSystemRootTokenxxx")
    assert_equal "new2xxxSystemRootTokenxxx", auth.api_token
    assert_equal User.find_by_uuid(system_user_uuid).id, auth.user_id
    # old token will fail
    assert_nil ApiClientAuthorization.validate(token: "newxxxSystemRootTokenxxx")
  end


end
