ENV["RAILS_ENV"] = "test"
unless ENV["NO_COVERAGE_TEST"]
  require 'simplecov'
  SimpleCov.start
end

require File.expand_path('../../config/environment', __FILE__)
require 'rails/test_help'

module ArvadosTestSupport
  def json_response
    ActiveSupport::JSON.decode @response.body
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
