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
require 'mocha/mini_test'

class ActiveSupport::TestCase
  # Setup all fixtures in test/fixtures/*.(yml|csv) for all tests in
  # alphabetical order.
  #
  # Note: You'll currently still have to declare fixtures explicitly
  # in integration tests -- they do not yet inherit this setting
  fixtures :all
  def use_token token_name
    auth = api_fixture('api_client_authorizations')[token_name.to_s]
    Thread.current[:arvados_api_token] = auth['api_token']
  end

  teardown do
    Thread.current[:arvados_api_token] = nil
    Thread.current[:user] = nil
    Thread.current[:reader_tokens] = nil
    # Diagnostics suite doesn't run a server, so there's no cache to clear.
    Rails.cache.clear unless (Rails.env == "diagnostics")
    # Restore configuration settings changed during tests
    $application_config.each do |k,v|
      if k.match /^[^.]*$/
        Rails.configuration.send (k + '='), v
      end
    end
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
        YAML.load(file)
      end
      keys.inject(@@api_fixtures[name]) { |hash, key| hash[key] }
    end
  end
  def api_fixture(name, *keys)
    self.class.api_fixture(name, *keys)
  end

  def find_fixture(object_class, name)
    object_class.find(api_fixture(object_class.to_s.pluralize.underscore,
                                  name, "uuid"))
  end
end

class ActiveSupport::TestCase
  include ApiFixtureLoader
  def session_for api_client_auth_name
    {
      arvados_api_token: api_fixture('api_client_authorizations')[api_client_auth_name.to_s]['api_token']
    }
  end
  def json_response
    Oj.load(@response.body)
  end
end

class ApiServerForTests
  ARV_API_SERVER_DIR = File.expand_path('../../../../services/api', __FILE__)
  SERVER_PID_PATH = File.expand_path('tmp/pids/wbtest-server.pid', ARV_API_SERVER_DIR)
  WEBSOCKET_PID_PATH = File.expand_path('tmp/pids/wstest-server.pid', ARV_API_SERVER_DIR)
  @main_process_pid = $$

  def _system(*cmd)
    $stderr.puts "_system #{cmd.inspect}"
    Bundler.with_clean_env do
      if not system({'RAILS_ENV' => 'test', "ARVADOS_WEBSOCKETS" => (if @websocket then "ws-only" end)}, *cmd)
        raise RuntimeError, "#{cmd[0]} returned exit code #{$?.exitstatus}"
      end
    end
  end

  def make_ssl_cert
    unless File.exists? './self-signed.key'
      _system('openssl', 'req', '-new', '-x509', '-nodes',
              '-out', './self-signed.pem',
              '-keyout', './self-signed.key',
              '-days', '3650',
              '-subj', '/CN=localhost')
    end
  end

  def kill_server
    if (pid = find_server_pid)
      $stderr.puts "Sending TERM to API server, pid #{pid}"
      Process.kill 'TERM', pid
    end
  end

  def find_server_pid
    pid = nil
    begin
      pid = IO.read(@pidfile).to_i
      $stderr.puts "API server is running, pid #{pid.inspect}"
    rescue Errno::ENOENT
    end
    return pid
  end

  def run(args=[])
    ::MiniTest.after_run do
      self.kill_server
    end

    @websocket = args.include?("--websockets")

    @pidfile = if @websocket
                 WEBSOCKET_PID_PATH
               else
                 SERVER_PID_PATH
               end

    # Kill server left over from previous test run
    self.kill_server

    Capybara.javascript_driver = :poltergeist
    Dir.chdir(ARV_API_SERVER_DIR) do |apidir|
      ENV["NO_COVERAGE_TEST"] = "1"
      if @websocket
        _system('bundle', 'exec', 'passenger', 'start', '-d', '-p3333',
                '--pid-file', @pidfile)
      else
        make_ssl_cert
        if ENV['ARVADOS_TEST_API_INSTALLED'].blank?
          _system('bundle', 'exec', 'rake', 'db:test:load')
          _system('bundle', 'exec', 'rake', 'db:fixtures:load')
        end
        _system('bundle', 'exec', 'passenger', 'start', '-d', '-p3000',
                '--pid-file', @pidfile,
                '--ssl',
                '--ssl-certificate', 'self-signed.pem',
                '--ssl-certificate-key', 'self-signed.key')
      end
      timeout = Time.now.tv_sec + 10
      good_pid = false
      while (not good_pid) and (Time.now.tv_sec < timeout)
        sleep 0.2
        server_pid = find_server_pid
        good_pid = (server_pid and
                    (server_pid > 0) and
                    (Process.kill(0, server_pid) rescue false))
      end
      if not good_pid
        raise RuntimeError, "could not find API server Rails pid"
      end
    end
  end

  def run_rake_task(task_name, arg_string)
    Dir.chdir(ARV_API_SERVER_DIR) do
      _system('bundle', 'exec', 'rake', "#{task_name}[#{arg_string}]")
    end
  end
end

class ActionController::TestCase
  setup do
    @counter = 0
  end

  def check_counter action
    @counter += 1
    if @counter == 2
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
    if self.class.want_reset_api_fixtures[:after_each_test]
      self.class.reset_api_fixtures_now
    end
    super
  end

  protected
  def self.reset_api_fixtures_now
    # Never try to reset fixtures when we're just using test
    # infrastructure to run performance/diagnostics suites.
    return unless Rails.env == 'test'

    auth = api_fixture('api_client_authorizations')['admin_trustedclient']
    Thread.current[:arvados_api_token] = auth['api_token']
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
