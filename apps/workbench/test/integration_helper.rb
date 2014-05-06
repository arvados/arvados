require 'test_helper'
require 'capybara/rails'
require 'capybara/poltergeist'
require 'uri'
require 'yaml'

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

class ActionDispatch::IntegrationTest
  # Make the Capybara DSL available in all integration tests
  include Capybara::DSL
  include ApiFixtureLoader
  include WaitForAjax

  @@API_AUTHS = self.api_fixture('api_client_authorizations')

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
end
