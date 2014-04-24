ENV["RAILS_ENV"] = "test"
require File.expand_path('../../config/environment', __FILE__)
require 'rails/test_help'

class ActiveSupport::TestCase
  # Setup all fixtures in test/fixtures/*.(yml|csv) for all tests in alphabetical order.
  #
  # Note: You'll currently still have to declare fixtures explicitly in integration tests
  # -- they do not yet inherit this setting
  fixtures :all

  teardown do
    Thread.current[:api_client_ip_address] = nil
    Thread.current[:api_client_authorization] = nil
    Thread.current[:api_client_uuid] = nil
    Thread.current[:api_client] = nil
    Thread.current[:user] = nil
  end

  def expect_json
    self.request.headers["Accept"] = "text/json"
  end

  def authorize_with(api_client_auth_name)
    self.request.env['HTTP_AUTHORIZATION'] = "OAuth2 #{api_client_authorizations(api_client_auth_name).api_token}"
  end

  # Add more helper methods to be used by all tests here...
end

class ActionDispatch::IntegrationTest

  teardown do
    Thread.current[:api_client_ip_address] = nil
    Thread.current[:api_client_authorization] = nil
    Thread.current[:api_client_uuid] = nil
    Thread.current[:api_client] = nil
    Thread.current[:user] = nil
  end

  def jresponse
    @jresponse ||= ActiveSupport::JSON.decode @response.body
  end

  def auth auth_fixture
    {'HTTP_AUTHORIZATION' => "OAuth2 #{api_client_authorizations(auth_fixture).api_token}"}
  end
end

# Ensure permissions are computed from the test fixtures.
User.invalidate_permissions_cache
