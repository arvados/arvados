require 'test_helper'
require 'capybara/rails'
require 'capybara/poltergeist'
require 'uri'
require 'yaml'

Capybara.register_driver :poltergeist do |app|
  Capybara::Poltergeist::Driver.new app, {
    window_size: [1200, 800],
    phantomjs_options: ['--ignore-ssl-errors=true'],
    inspector: true,
  }
end

module WaitForAjax
  Capybara.default_wait_time = 5
  def wait_for_ajax
    Timeout.timeout(Capybara.default_wait_time) do
      loop until finished_all_ajax_requests?
    end
  end

  def finished_all_ajax_requests?
    page.evaluate_script('jQuery.active').zero?
  end
end

module AssertDomEvent
  # Yield the supplied block, then wait for an event to arrive at a
  # DOM element.
  def assert_triggers_dom_event events, target='body'
    magic = 'received-dom-event-' + rand(2**30).to_s(36)
    page.evaluate_script <<eos
      $('#{target}').one('#{events}', function() {
        $('body').addClass('#{magic}');
      });
eos
    yield
    assert_selector "body.#{magic}"
    page.evaluate_script "$('body').removeClass('#{magic}');";
  end
end

module HeadlessHelper
  class HeadlessSingleton
    def self.get
      @headless ||= Headless.new reuse: false
    end
  end

  Capybara.default_driver = :rack_test

  def self.included base
    base.class_eval do
      setup do
        Capybara.use_default_driver
        @headless = false
      end

      teardown do
        if @headless
          @headless.stop
          @headless = false
        end
      end
    end
  end

  def need_selenium reason=nil
    Capybara.current_driver = :selenium
    unless ENV['ARVADOS_TEST_HEADFUL'] or @headless
      @headless = HeadlessSingleton.get
      @headless.start
    end
  end

  def need_javascript reason=nil
    unless Capybara.current_driver == :selenium
      Capybara.current_driver = :poltergeist
    end
  end
end

class ActionDispatch::IntegrationTest
  # Make the Capybara DSL available in all integration tests
  include Capybara::DSL
  include ApiFixtureLoader
  include WaitForAjax
  include AssertDomEvent
  include HeadlessHelper

  @@API_AUTHS = self.api_fixture('api_client_authorizations')

  def page_with_token(token, path='/')
    # Generate a page path with an embedded API token.
    # Typical usage: visit page_with_token('token_name', page)
    # The token can be specified by the name of an api_client_authorizations
    # fixture, or passed as a raw string.
    api_token = ((@@API_AUTHS.include? token) ?
                 @@API_AUTHS[token]['api_token'] : token)
    path_parts = path.partition("#")
    sep = (path_parts.first.include? '?') ? '&' : '?'
    q_string = URI.encode_www_form('api_token' => api_token)
    path_parts.insert(1, "#{sep}#{q_string}")
    path_parts.join("")
  end

  # Find a page element, but return false instead of raising an
  # exception if not found. Use this with assertions to explain that
  # the error signifies a failed test rather than an unexpected error
  # during a testing procedure.
  def find? *args
    begin
      find *args
    rescue Capybara::ElementNotFound
      false
    end
  end

  @@screenshot_count = 1
  def screenshot
    image_file = "./tmp/workbench-fail-#{@@screenshot_count}.png"
    begin
      page.save_screenshot image_file
    rescue Capybara::NotSupportedByDriverError
      # C'est la vie.
    else
      puts "Saved #{image_file}"
      @@screenshot_count += 1
    end
  end

  teardown do
    if not passed?
      screenshot
    end
    if Capybara.current_driver == :selenium
      page.execute_script("window.localStorage.clear()")
    end
    Capybara.reset_sessions!
  end
end
