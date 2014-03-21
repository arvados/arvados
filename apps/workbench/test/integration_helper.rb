require 'test_helper'
require 'capybara/rails'
require 'capybara/poltergeist'
require 'uri'
require 'yaml'

$ARV_API_SERVER_DIR = File.expand_path('../../../../services/api', __FILE__)

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
  # Don't try to re-use the current Bundle environment when we launch the
  # API server.
  @@APIENV = {
    'BUNDLE_BIN_PATH' => nil,
    'BUNDLE_GEMFILE' => nil,
    'RUBYLIB' => nil,
    'RUBYOPT' => nil,
  }

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
      pid_path = 'tmp/pids/server.pid'
      timeout = Time.now.tv_sec + 5
      while (not File.exists? pid_path) and (Time.now.tv_sec < timeout)
        sleep 0.2
      end
      IO.read(pid_path).to_i
    end
    begin
      super(args)
    ensure
      Process.kill('TERM', server_pid)
    end
  end
end

MiniTest::Unit.runner = IntegrationTestRunner.new
