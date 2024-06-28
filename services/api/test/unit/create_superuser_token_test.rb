# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

require 'safe_json'
require 'test_helper'
require 'create_superuser_token'

class CreateSuperUserTokenTest < ActiveSupport::TestCase
  include CreateSuperUserToken

  test "create superuser token twice and expect same results" do
    # Create a token with some string
    token1 = create_superuser_token 'atesttoken'
    assert_not_nil token1
    assert_match(/atesttoken$/, token1)

    # Create token again; this time, we should get the one created earlier
    token2 = create_superuser_token
    assert_not_nil token2
    assert_equal token1, token2
  end

  test "create superuser token with two different inputs and expect the first both times" do
    # Create a token with some string
    token1 = create_superuser_token 'atesttoken'
    assert_not_nil token1
    assert_match(/\/atesttoken$/, token1)

    # Create token again with some other string and expect the existing superuser token back
    token2 = create_superuser_token 'someothertokenstring'
    assert_not_nil token2
    assert_equal token1, token2
  end

  test "create superuser token and invoke again with some other valid token" do
    # Create a token with some string
    token1 = create_superuser_token 'atesttoken'
    assert_not_nil token1
    assert_match(/\/atesttoken$/, token1)

    su_token = api_client_authorizations("system_user").api_token
    token2 = create_superuser_token su_token
    assert_equal token2.split('/')[2], su_token
  end

  test "create superuser token, expire it, and create again" do
    ApiClientAuthorization.where(user_id: system_user.id).delete_all

    # Create a token with some string
    token1 = create_superuser_token 'atesttoken'
    assert_not_nil token1
    assert_match(/\/atesttoken$/, token1)

    # Expire this token and call create again; expect a new token created
    apiClientAuth = ApiClientAuthorization.where(api_token: 'atesttoken').first
    refute_nil apiClientAuth
    Thread.current[:user] = users(:admin)
    apiClientAuth.update expires_at: '2000-10-10'

    token2 = create_superuser_token
    assert_not_nil token2
    assert_not_equal token1, token2
  end

  test "invoke create superuser token with an invalid non-superuser token and expect error" do
    active_user_token = api_client_authorizations("active").api_token
    e = assert_raises RuntimeError do
      create_superuser_token active_user_token
    end
    assert_not_nil e
    assert_equal "Token exists but is not a superuser token.", e.message
  end

  test "specified token has limited scope" do
    active_user_token = api_client_authorizations("data_manager").api_token
    e = assert_raises RuntimeError do
      create_superuser_token active_user_token
    end
    assert_not_nil e
    assert_match /^Token exists but has limited scope/, e.message
  end

  test "existing token has limited scope" do
    active_user_token = api_client_authorizations("admin_vm").api_token
    ApiClientAuthorization.
      where(user_id: system_user.id).
      update_all(scopes: ["GET /"])
    fixture_tokens = ApiClientAuthorization.all.collect(&:api_token)
    new_token = create_superuser_token
    refute_includes(fixture_tokens, new_token)
  end
end
