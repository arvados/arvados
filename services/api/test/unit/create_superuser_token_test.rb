require 'test_helper'
require 'create_superuser_token'

class CreateSuperUserTokenTest < ActiveSupport::TestCase
  include CreateSuperUserToken

  test "create superuser token twice and expect same resutls" do
    # Create a token with some string
    token1 = create_superuser_token 'atesttoken'
    assert_not_nil token1
    assert_equal token1, 'atesttoken'

    # Create token again; this time, we should get the one created earlier
    token2 = create_superuser_token
    assert_not_nil token2
    assert_equal token1, token2
  end

  test "create superuser token with two different inputs and expect the first both times" do
    # Create a token with some string
    token1 = create_superuser_token 'atesttoken'
    assert_not_nil token1
    assert_equal token1, 'atesttoken'

    # Create token again with some other string and expect the existing superuser token back
    token2 = create_superuser_token 'someothertokenstring'
    assert_not_nil token2
    assert_equal token1, token2
  end

  test "create superuser token twice and expect same results" do
    # Create a token with some string
    token1 = create_superuser_token 'atesttoken'
    assert_not_nil token1
    assert_equal token1, 'atesttoken'

    # Create token again with that same superuser token and expect it back
    token2 = create_superuser_token 'atesttoken'
    assert_not_nil token2
    assert_equal token1, token2
  end

  test "create superuser token and invoke again with some other valid token" do
    # Create a token with some string
    token1 = create_superuser_token 'atesttoken'
    assert_not_nil token1
    assert_equal token1, 'atesttoken'

    su_token = api_client_authorizations("system_user").api_token
    token2 = create_superuser_token su_token
    assert_equal token2, su_token
  end

  test "create superuser token, expire it, and create again" do
    # Create a token with some string
    token1 = create_superuser_token 'atesttoken'
    assert_not_nil token1
    assert_equal token1, 'atesttoken'

    # Expire this token and call create again; expect a new token created
    apiClientAuth = ApiClientAuthorization.where(api_token: token1).first
    Thread.current[:user] = users(:admin)
    apiClientAuth.update_attributes expires_at: '2000-10-10'

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
    assert_equal "Token already exists but is not a superuser token.", e.message
  end
end
