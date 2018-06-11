# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

require 'test_helper'
require 'capybara/rails'
require 'capybara/poltergeist'
require 'uri'
require 'yaml'

def available_port for_what
  begin
    Addrinfo.tcp("0.0.0.0", 0).listen do |srv|
      port = srv.connect_address.ip_port
      # Selenium needs an additional locking port, check if it's available
      # and retry if necessary.
      if for_what == 'selenium'
        locking_port = port - 1
        Addrinfo.tcp("0.0.0.0", locking_port).listen.close
      end
      STDERR.puts "Using port #{port} for #{for_what}"
      return port
    end
  rescue Errno::EADDRINUSE, Errno::EACCES
    retry
  end
end

def selenium_opts
  {
    port: available_port('selenium'),
  }
end

def poltergeist_opts
  {
    phantomjs_options: ['--ignore-ssl-errors=true'],
    port: available_port('poltergeist'),
    window_size: [1200, 800],
  }
end

Capybara.register_driver :poltergeist do |app|
  Capybara::Poltergeist::Driver.new app, poltergeist_opts
end

Capybara.register_driver :poltergeist_debug do |app|
  Capybara::Poltergeist::Driver.new app, poltergeist_opts.merge(inspector: true)
end

Capybara.register_driver :poltergeist_with_fake_websocket do |app|
  js = File.expand_path '../support/fake_websocket.js', __FILE__
  Capybara::Poltergeist::Driver.new app, poltergeist_opts.merge(extensions: [js])
end

Capybara.register_driver :poltergeist_without_file_api do |app|
  js = File.expand_path '../support/remove_file_api.js', __FILE__
  Capybara::Poltergeist::Driver.new app, poltergeist_opts.merge(extensions: [js])
end

Capybara.register_driver :selenium do |app|
  Capybara::Selenium::Driver.new app, selenium_opts
end

Capybara.register_driver :selenium_with_download do |app|
  profile = Selenium::WebDriver::Firefox::Profile.new
  profile['browser.download.dir'] = DownloadHelper.path.to_s
  profile['browser.download.downloadDir'] = DownloadHelper.path.to_s
  profile['browser.download.defaultFolder'] = DownloadHelper.path.to_s
  profile['browser.download.folderList'] = 2 # "save to user-defined location"
  profile['browser.download.manager.showWhenStarting'] = false
  profile['browser.helperApps.alwaysAsk.force'] = false
  profile['browser.helperApps.neverAsk.saveToDisk'] = 'text/plain,application/octet-stream'
  Capybara::Selenium::Driver.new app, selenium_opts.merge(profile: profile)
end

module WaitForAjax
  # FIXME: Huge side effect here
  # The following line changes the global default Capybara wait time, affecting
  # every test which follows this one. This should be removed and the failing tests
  # should have their individual wait times increased, if appropriate, using
  # the using_wait_time(N) construct to temporarily change the wait time.
  # Note: the below is especially bad because there are places that increase wait
  # times using a multiplier e.g. using_wait_time(3 * Capybara.default_max_wait_time)
  Capybara.default_max_wait_time = 10
  def wait_for_ajax
    timeout = 10
    count = 0
    while page.evaluate_script("jQuery.active").to_i > 0
      count += 1
      raise "AJAX request took more than #{timeout} seconds" if count > timeout * 10
      sleep(0.1)
    end
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
    @display = ENV['ARVADOS_TEST_HEADLESS_DISPLAY'] || rand(400)+100
    STDERR.puts "Using display :#{@display} for headless tests"
    def self.get
      @headless ||= Headless.new reuse: false, display: @display
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

  def need_selenium reason=nil, driver=:selenium
    Capybara.current_driver = driver
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

module KeepWebConfig
  def getport service
    File.read(File.expand_path("../../../../tmp/#{service}.port", __FILE__))
  end

  def use_keep_web_config
    @kwport = getport 'keep-web-ssl'
    @kwdport = getport 'keep-web-dl-ssl'
    Rails.configuration.keep_web_url = "https://localhost:#{@kwport}/c=%{uuid_or_pdh}"
    Rails.configuration.keep_web_download_url = "https://localhost:#{@kwdport}/c=%{uuid_or_pdh}"
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
    else
      page.driver.restart if defined?(page.driver.restart)
    end
    Capybara.reset_sessions!
  end

  def accept_alert
    if Capybara.current_driver == :selenium
      (0..9).each do
        begin
          page.driver.browser.switch_to.alert.accept
          break
        rescue Selenium::WebDriver::Error::NoSuchAlertError
         sleep 0.1
        end
      end
    else
      # poltergeist returns true for confirm, so no need to accept
    end
  end
end

def upload_data_and_get_collection(data, user, filename, owner_uuid=nil)
  token = api_fixture('api_client_authorizations')[user]['api_token']
  datablock = `echo -n #{data.shellescape} | ARVADOS_API_TOKEN=#{token.shellescape} arv-put --no-progress --raw -`.strip
  assert $?.success?, $?
  col = nil
  use_token user do
    mtxt = ". #{datablock} 0:#{data.length}:#{filename}\n"
    if owner_uuid
      col = Collection.create(manifest_text: mtxt, owner_uuid: owner_uuid)
    else
      col = Collection.create(manifest_text: mtxt)
    end
  end
  return col
end
