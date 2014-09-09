ENV["RAILS_ENV"] = "test" if !ENV["RAILS_ENV"]

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

  def teardown
    Thread.current[:arvados_api_token] = nil
    Thread.current[:reader_tokens] = nil
    super
  end
end

module ApiFixtureLoader
  def self.included(base)
    base.extend(ClassMethods)
  end

  module ClassMethods
    @@api_fixtures = {}
    def api_fixture(name)
      # Returns the data structure from the named API server test fixture.
      @@api_fixtures[name] ||= \
      begin
        path = File.join(ApiServerForTests::ARV_API_SERVER_DIR,
                         'test', 'fixtures', "#{name}.yml")
        YAML.load(IO.read(path))
      end
    end
  end
  def api_fixture name
    self.class.api_fixture name
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
  @main_process_pid = $$

  def self._system(*cmd)
    $stderr.puts "_system #{cmd.inspect}"
    Bundler.with_clean_env do
      if not system({'RAILS_ENV' => 'test'}, *cmd)
        raise RuntimeError, "#{cmd[0]} returned exit code #{$?.exitstatus}"
      end
    end
  end

  def self.make_ssl_cert
    unless File.exists? './self-signed.key'
      _system('openssl', 'req', '-new', '-x509', '-nodes',
              '-out', './self-signed.pem',
              '-keyout', './self-signed.key',
              '-days', '3650',
              '-subj', '/CN=localhost')
    end
  end

  def self.kill_server
    if (pid = find_server_pid)
      $stderr.puts "Sending TERM to API server, pid #{pid}"
      Process.kill 'TERM', pid
    end
  end

  def self.find_server_pid
    pid = nil
    begin
      pid = IO.read(SERVER_PID_PATH).to_i
      $stderr.puts "API server is running, pid #{pid.inspect}"
    rescue Errno::ENOENT
    end
    return pid
  end

  def self.run(args=[])
    ::MiniTest.after_run do
      self.kill_server
    end

    # Kill server left over from previous test run
    self.kill_server

    Capybara.javascript_driver = :poltergeist
    Dir.chdir(ARV_API_SERVER_DIR) do |apidir|
      ENV["NO_COVERAGE_TEST"] = "1"
      make_ssl_cert
      _system('bundle', 'exec', 'rake', 'db:test:load')
      _system('bundle', 'exec', 'rake', 'db:fixtures:load')
      _system('bundle', 'exec', 'passenger', 'start', '-d', '-p3001',
              '--pid-file', SERVER_PID_PATH,
              '--ssl',
              '--ssl-certificate', 'self-signed.pem',
              '--ssl-certificate-key', 'self-signed.key')
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
end

if ENV["RAILS_ENV"].eql? 'test'
  ApiServerForTests.run
end
