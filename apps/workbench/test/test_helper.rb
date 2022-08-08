# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

ENV["RAILS_ENV"] = "test" if (ENV["RAILS_ENV"] != "diagnostics" and ENV["RAILS_ENV"] != "performance")

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
    end
  rescue Exception => e
    $stderr.puts "SimpleCov unavailable (#{e}). Proceeding without."
  end
end

require File.expand_path('../../config/environment', __FILE__)
require 'rails/test_help'
require 'mocha/minitest'

class ActiveSupport::TestCase
  # Setup all fixtures in test/fixtures/*.(yml|csv) for all tests in
  # alphabetical order.
  #
  # Note: You'll currently still have to declare fixtures explicitly
  # in integration tests -- they do not yet inherit this setting
  fixtures :all
  def use_token(token_name)
    user_was = Thread.current[:user]
    token_was = Thread.current[:arvados_api_token]
    auth = api_fixture('api_client_authorizations')[token_name.to_s]
    Thread.current[:arvados_api_token] = "v2/#{auth['uuid']}/#{auth['api_token']}"
    if block_given?
      begin
        yield
      ensure
        Thread.current[:user] = user_was
        Thread.current[:arvados_api_token] = token_was
      end
    end
  end

  teardown do
    Thread.current[:arvados_api_token] = nil
    Thread.current[:user] = nil
    Thread.current[:reader_tokens] = nil
    # Diagnostics suite doesn't run a server, so there's no cache to clear.
    Rails.cache.clear unless (Rails.env == "diagnostics")
    # Restore configuration settings changed during tests
    self.class.reset_application_config
  end

  def self.reset_application_config
    # Restore configuration settings changed during tests
    ConfigLoader.copy_into_config $arvados_config, Rails.configuration
    ConfigLoader.copy_into_config $remaining_config, Rails.configuration
    Rails.configuration.Services.Controller.ExternalURL = URI("https://#{ENV['ARVADOS_API_HOST']}")
    Rails.configuration.TLS.Insecure = true
  end
end

module ApiFixtureLoader
  def self.included(base)
    base.extend(ClassMethods)
  end

  module ClassMethods
    @@api_fixtures = {}
    def api_fixture(name, *keys)
      # Returns the data structure from the named API server test fixture.
      @@api_fixtures[name] ||= \
      begin
        path = File.join(ApiServerForTests::ARV_API_SERVER_DIR,
                         'test', 'fixtures', "#{name}.yml")
        file = IO.read(path)
        trim_index = file.index('# Test Helper trims the rest of the file')
        file = file[0, trim_index] if trim_index
        YAML.load(file).each do |name, ob|
          ob.reject! { |k, v| k.start_with?('secret_') }
        end
      end
      keys.inject(@@api_fixtures[name]) { |hash, key| hash[key] }.deep_dup
    end
  end

  def api_fixture(name, *keys)
    self.class.api_fixture(name, *keys)
  end

  def api_token(name)
    auth = api_fixture('api_client_authorizations')[name]
    "v2/#{auth['uuid']}/#{auth['api_token']}"
  end

  def find_fixture(object_class, name)
    object_class.find(api_fixture(object_class.to_s.pluralize.underscore,
                                  name, "uuid"))
  end
end

module ApiMockHelpers
  def fake_api_response body, status_code, headers
    resp = mock
    resp.responds_like_instance_of HTTP::Message
    resp.stubs(:headers).returns headers
    resp.stubs(:content).returns body
    resp.stubs(:status_code).returns status_code
    resp
  end

  def stub_api_calls_with_body body, status_code=200, headers={}
    stub_api_calls
    resp = fake_api_response body, status_code, headers
    stub_api_client.stubs(:post).returns resp
  end

  def stub_api_calls
    @stubbed_client = ArvadosApiClient.new
    @stubbed_client.instance_eval do
      @api_client = HTTPClient.new
    end
    ArvadosApiClient.stubs(:new_or_current).returns(@stubbed_client)
  end

  def stub_api_calls_with_invalid_json
    stub_api_calls_with_body ']"omg,bogus"['
  end

  # Return the HTTPClient mock used by the ArvadosApiClient mock. You
  # must have called stub_api_calls first.
  def stub_api_client
    @stubbed_client.instance_eval do
      @api_client
    end
  end
end

class ActiveSupport::TestCase
  include ApiMockHelpers
end

class ActiveSupport::TestCase
  include ApiFixtureLoader
  def session_for api_client_auth_name
    auth = api_fixture('api_client_authorizations')[api_client_auth_name.to_s]
    {
      arvados_api_token: "v2/#{auth['uuid']}/#{auth['api_token']}"
    }
  end
  def json_response
    Oj.safe_load(@response.body)
  end
end

class ApiServerForTests
  PYTHON_TESTS_DIR = File.expand_path('../../../../sdk/python/tests', __FILE__)
  ARV_API_SERVER_DIR = File.expand_path('../../../../services/api', __FILE__)
  SERVER_PID_PATH = File.expand_path('tmp/pids/test-server.pid', ARV_API_SERVER_DIR)
  WEBSOCKET_PID_PATH = File.expand_path('tmp/pids/test-server.pid', ARV_API_SERVER_DIR)
  @main_process_pid = $$
  @@server_is_running = false

  def check_output *args
    output = nil
    Bundler.with_clean_env do
      output = IO.popen *args do |io|
        io.read
      end
      if not $?.success?
        raise RuntimeError, "Command failed (#{$?}): #{args.inspect}"
      end
    end
    output
  end

  def run_test_server
    Dir.chdir PYTHON_TESTS_DIR do
      check_output %w(python ./run_test_server.py start_keep)
    end
  end

  def stop_test_server
    Dir.chdir PYTHON_TESTS_DIR do
      check_output %w(python ./run_test_server.py stop_keep)
    end
    @@server_is_running = false
  end

  def run args=[]
    return if @@server_is_running

    # Stop server left over from interrupted previous run
    stop_test_server

    ::MiniTest.after_run do
      stop_test_server
    end

    run_test_server
    ActiveSupport::TestCase.reset_application_config

    @@server_is_running = true
  end

  def run_rake_task task_name, arg_string
    Dir.chdir ARV_API_SERVER_DIR do
      check_output ['bundle', 'exec', 'rake', "#{task_name}[#{arg_string}]"]
    end
  end
end

class ActionController::TestCase
  setup do
    @test_counter = 0
  end

  def check_counter action
    @test_counter += 1
    if @test_counter == 2
      assert_equal 1, 2, "Multiple actions in controller test"
    end
  end

  [:get, :post, :put, :patch, :delete].each do |method|
    define_method method do |action, *args|
      check_counter action
      super action, *args
    end
  end
end

# Test classes can call reset_api_fixtures(when_to_reset,flag) to
# override the default. Example:
#
# class MySuite < ActionDispatch::IntegrationTest
#   reset_api_fixtures :after_each_test, false
#   reset_api_fixtures :after_suite, true
#   ...
# end
#
# The default behavior is reset_api_fixtures(:after_each_test,true).
#
class ActiveSupport::TestCase

  def self.inherited subclass
    subclass.class_eval do
      class << self
        attr_accessor :want_reset_api_fixtures
      end
      @want_reset_api_fixtures = {
        after_each_test: true,
        after_suite: false,
        before_suite: false,
      }
    end
    super
  end
  # Existing subclasses of ActiveSupport::TestCase (ones that already
  # existed before we set up the self.inherited hook above) will not
  # get their own instance variable. They're not real test cases
  # anyway, so we give them a "don't reset anywhere" stub.
  def self.want_reset_api_fixtures
    {}
  end

  def self.reset_api_fixtures where, t=true
    if not want_reset_api_fixtures.has_key? where
      raise ArgumentError, "There is no #{where.inspect} hook"
    end
    self.want_reset_api_fixtures[where] = t
  end

  def self.run *args
    reset_api_fixtures_now if want_reset_api_fixtures[:before_suite]
    result = super
    reset_api_fixtures_now if want_reset_api_fixtures[:after_suite]
    result
  end

  def after_teardown
    if self.class.want_reset_api_fixtures[:after_each_test] and
        (!defined?(@want_reset_api_fixtures) or @want_reset_api_fixtures != false)
      self.class.reset_api_fixtures_now
    end
    super
  end

  def reset_api_fixtures_after_test t=true
    @want_reset_api_fixtures = t
  end

  protected
  def self.reset_api_fixtures_now
    # Never try to reset fixtures when we're just using test
    # infrastructure to run performance/diagnostics suites.
    return unless Rails.env == 'test'

    auth = api_fixture('api_client_authorizations')['admin_trustedclient']
    Thread.current[:arvados_api_token] = "v2/#{auth['uuid']}/#{auth['api_token']}"
    ArvadosApiClient.new.api(nil, '../../database/reset', {})
    Thread.current[:arvados_api_token] = nil
  end
end

# If it quacks like a duck, it must be a HTTP request object.
class RequestDuck
  def self.host
    "localhost"
  end

  def self.port
    8080
  end

  def self.protocol
    "http"
  end
end

# Example:
#
# apps/workbench$ RAILS_ENV=test bundle exec irb -Ilib:test
# > load 'test/test_helper.rb'
# > singletest 'integration/collection_upload_test.rb', 'Upload two empty files'
#
def singletest test_class_file, test_name
  load File.join('test', test_class_file)
  Minitest.run ['-v', '-n', "test_#{test_name.gsub ' ', '_'}"]
  Object.send(:remove_const,
              test_class_file.gsub(/.*\/|\.rb$/, '').camelize.to_sym)
  ::Minitest::Runnable.runnables.reject! { true }
end

if ENV["RAILS_ENV"].eql? 'test'
  ApiServerForTests.new.run
  ApiServerForTests.new.run ["--websockets"]
end

# Reset fixtures now (i.e., before any tests run).
ActiveSupport::TestCase.reset_api_fixtures_now

module Minitest
  class Test
    def capture_exceptions *args
      begin
        n = 0
        begin
          yield
        rescue *PASSTHROUGH_EXCEPTIONS
          raise
        rescue Exception => e
          n += 1
          raise if n > 2 || e.is_a?(Skip)
          STDERR.puts "Test failed, retrying (##{n})"
          ActiveSupport::TestCase.reset_api_fixtures_now
          retry
        end
      rescue *PASSTHROUGH_EXCEPTIONS
        raise
      rescue Assertion => e
        self.failures << e
      rescue Exception => e
        self.failures << UnexpectedError.new(e)
      end
    end
  end
end
