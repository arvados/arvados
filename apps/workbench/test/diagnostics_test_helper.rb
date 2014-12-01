require 'integration_helper'
require 'yaml'

# Diagnostics tests are executed when "RAILS_ENV=diagnostics" is used.
# When "RAILS_ENV=test" is used, tests in the "diagnostics" directory
# will not be executed.

class DiagnosticsTest < ActionDispatch::IntegrationTest

  # Prepends workbench URL to the path provided and visits that page
  # Expects path parameters such as "/collections/<uuid>"
  def visit_page_with_token token_name, path='/'
    workbench_url = Rails.configuration.arvados_workbench_url
    if workbench_url.end_with? '/'
      workbench_url = workbench_url[0, workbench_url.size-1]
    end
    tokens = Rails.configuration.user_tokens
    visit page_with_token(tokens[token_name], (workbench_url + path))
  end

  # Looks for the text_to_look_for for up to the max_time provided
  def wait_until_page_has text_to_look_for, max_time=30
    max_time = 30 if (!max_time || (max_time.to_s != max_time.to_i.to_s))
    Timeout.timeout(max_time) do
      loop until page.has_text?(text_to_look_for)
    end
  end

  # Diagnostics tests never want to reset the database fixtures.
  protected
  def self.reset_api_fixtures_now end

end
