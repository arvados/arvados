require 'test_helper'
require 'capybara/rails'
require 'capybara/poltergeist'
require 'uri'
require 'yaml'

$ARV_API_SERVER_DIR = File.expand_path('../../../../services/api', __FILE__)
SERVER_PID_PATH = 'tmp/pids/server.pid'

class ActionDispatch::IntegrationTest
  # Make the Capybara DSL available in all integration tests
  include Capybara::DSL

  def self.api_fixture(name)
    # Returns the data structure from the named API server test fixture.
    path = File.join($ARV_API_SERVER_DIR, 'test', 'fixtures', "#{name}.yml")
    YAML.load(IO.read(path))
  end

  @@API_AUTHS = api_fixture('api_client_authorizations')

  def page_with_token(token, path='/')
    # Generate a page path with an embedded API token.
    # Typical usage: visit page_with_token('token_name', page)
    # The token can be specified by the name of an api_client_authorizations
    # fixture, or passed as a raw string.
    api_token = ((@@API_AUTHS.include? token) ?
                 @@API_AUTHS[token]['api_token'] : token)
    sep = (path.include? '?') ? '&' : '?'
    q_string = URI.encode_www_form('api_token' => api_token)
    "#{path}#{sep}#{q_string}"
  end
end

class IntegrationTestRunner < MiniTest::Unit
  # Make a hash that unsets Bundle's environment variables.
  # We'll use this environment when we launch Bundle commands in the API
  # server.  Otherwise, those commands will try to use Workbench's gems, etc.
  @@APIENV = ENV.map { |(key, val)| (key =~ /^BUNDLE_/) ? [key, nil] : nil }.
    compact.to_h

  def _system(*cmd)
    if not system(@@APIENV, *cmd)
      raise RuntimeError, "#{cmd[0]} returned exit code #{$?.exitstatus}"
    end
  end

  def _run(args=[])
    Capybara.javascript_driver = :poltergeist
    server_pid = Dir.chdir($ARV_API_SERVER_DIR) do |apidir|
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

MiniTest::Unit.runner = IntegrationTestRunner.new
