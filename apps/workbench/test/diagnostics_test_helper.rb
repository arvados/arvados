require 'integration_helper'
require 'yaml'

class DiagnosticsTest < ActionDispatch::IntegrationTest

  def visit_page_with_token token_name, path='/'
    if !path.start_with? Rails.configuration.arvados_workbench_url
      path = Rails.configuration.arvados_workbench_url + path
    end
    tokens = Rails.configuration.diagnostics_testing_user_tokens
    visit page_with_token(tokens[token_name], path)
  end

  def wait_until_page_has text_to_look_for, max_time=30
    max_time = 30 if (!max_time || (max_time.to_s != max_time.to_i.to_s))
    Timeout.timeout(max_time) do
      loop until page.has_text?(text_to_look_for)
    end
  end

end
