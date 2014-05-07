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
    end
  rescue Exception => e
    $stderr.puts "SimpleCov unavailable (#{e}). Proceeding without."
  end
end

require File.expand_path('../../config/environment', __FILE__)
require 'rails/test_help'

$ARV_API_SERVER_DIR = File.expand_path('../../../../services/api', __FILE__)
SERVER_PID_PATH = 'tmp/pids/server.pid'

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
        path = File.join($ARV_API_SERVER_DIR, 'test', 'fixtures', "#{name}.yml")
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
end

class ApiServerBackedTestRunner < MiniTest::Unit
  # Make a hash that unsets Bundle's environment variables.
  # We'll use this environment when we launch Bundle commands in the API
  # server.  Otherwise, those commands will try to use Workbench's gems, etc.
  @@APIENV = Hash[ENV.map { |key, val|
                    (key =~ /^BUNDLE_/) ? [key, nil] : nil
                  }.compact]

  def _system(*cmd)
    if not system(@@APIENV, *cmd)
      raise RuntimeError, "#{cmd[0]} returned exit code #{$?.exitstatus}"
    end
  end

  def _run(args=[])
    Capybara.javascript_driver = :poltergeist
    server_pid = Dir.chdir($ARV_API_SERVER_DIR) do |apidir|
      ENV["NO_COVERAGE_TEST"] = "1"
      _system('bundle', 'exec', 'rake', 'db:test:load')
      _system('bundle', 'exec', 'rake', 'db:fixtures:load')
      _system('bundle', 'exec', 'rails', 'server', '-d')
      timeout = Time.now.tv_sec + 10
      begin
        sleep 0.2
        begin
          server_pid = IO.read(SERVER_PID_PATH).to_i
          good_pid = (server_pid > 0) and (Process.kill(0, pid) rescue false)
        rescue Errno::ENOENT
          good_pid = false
        end
      end while (not good_pid) and (Time.now.tv_sec < timeout)
      if not good_pid
        raise RuntimeError, "could not find API server Rails pid"
      end
      server_pid
    end
    begin
      super(args)
    ensure
      Process.kill('TERM', server_pid)
    end
  end
end

MiniTest::Unit.runner = ApiServerBackedTestRunner.new
