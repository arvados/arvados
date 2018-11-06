# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

require 'test_helper'
require 'sweep_trashed_objects'

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

  test "delete expired in SweepTrashedObjects" do
    assert_not_empty ApiClientAuthorization.where(uuid: api_client_authorizations(:expired).uuid)
    SweepTrashedObjects.sweep_now
    assert_empty ApiClientAuthorization.where(uuid: api_client_authorizations(:expired).uuid)
  end

end
