ENV["RAILS_ENV"] = "test"
require File.expand_path('../../config/environment', __FILE__)
require 'rails/test_help'

module ArvadosTestSupport
  def json_response
    @json_response ||= ActiveSupport::JSON.decode @response.body
  end

  def api_token(api_client_auth_name)
    api_client_authorizations(api_client_auth_name).api_token
  end

  def auth(api_client_auth_name)
    {'HTTP_AUTHORIZATION' => "OAuth2 #{api_token(api_client_auth_name)}"}
  end
end

class ActiveSupport::TestCase
  # Setup all fixtures in test/fixtures/*.(yml|csv) for all tests in alphabetical order.
  #
  # Note: You'll currently still have to declare fixtures explicitly in integration tests
  # -- they do not yet inherit this setting
  fixtures :all

  include ArvadosTestSupport

  teardown do
    Thread.current[:api_client_ip_address] = nil
    Thread.current[:api_client_authorization] = nil
    Thread.current[:api_client_uuid] = nil
    Thread.current[:api_client] = nil
    Thread.current[:user] = nil
  end

  def set_user_from_auth(auth_name)
    client_auth = api_client_authorizations(auth_name)
    Thread.current[:api_client_authorization] = client_auth
    Thread.current[:api_client] = client_auth.api_client
    Thread.current[:user] = client_auth.user
  end

  def expect_json
    self.request.headers["Accept"] = "text/json"
  end

  def authorize_with(api_client_auth_name)
    ArvadosApiToken.new.call ({"rack.input" => "", "HTTP_AUTHORIZATION" => "OAuth2 #{api_client_authorizations(api_client_auth_name).api_token}"})
  end
end

class ActionDispatch::IntegrationTest
  teardown do
    Thread.current[:api_client_ip_address] = nil
    Thread.current[:api_client_authorization] = nil
    Thread.current[:api_client_uuid] = nil
    Thread.current[:api_client] = nil
    Thread.current[:user] = nil
  end
end

# Ensure permissions are computed from the test fixtures.
User.invalidate_permissions_cache
