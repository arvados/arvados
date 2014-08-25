ENV["RAILS_ENV"] = "test"
unless ENV["NO_COVERAGE_TEST"]
  begin
    require 'simplecov'
    require 'simplecov-rcov'
    class SimpleCov::Formatter::MergedFormatter
      def format(result)
        SimpleCov::Formatter::HTMLFormatter.new.format(result)
        SimpleCov::Formatter::RcovFormatter.new.format(result)
      end
    end
    SimpleCov.formatter = SimpleCov::Formatter::MergedFormatter
    SimpleCov.start do
      add_filter '/test/'
      add_filter 'initializers/secret_token'
      add_filter 'initializers/omniauth'
    end
  rescue Exception => e
    $stderr.puts "SimpleCov unavailable (#{e}). Proceeding without."
  end
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
  include FactoryGirl::Syntax::Methods
  fixtures :all

  include ArvadosTestSupport

  teardown do
    Thread.current[:api_client_ip_address] = nil
    Thread.current[:api_client_authorization] = nil
    Thread.current[:api_client_uuid] = nil
    Thread.current[:api_client] = nil
    Thread.current[:user] = nil
    # Restore configuration settings changed during tests
    $application_config.each do |k,v|
      if k.match /^[^.]*$/
        Rails.configuration.send (k + '='), v
      end
    end
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

  def authorize_with api_client_auth_name
    authorize_with_token api_client_authorizations(api_client_auth_name).api_token
  end

  def authorize_with_token token
    t = token
    t = t.api_token if t.respond_to? :api_token
    ArvadosApiToken.new.call("rack.input" => "",
                             "HTTP_AUTHORIZATION" => "OAuth2 #{t}")
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
