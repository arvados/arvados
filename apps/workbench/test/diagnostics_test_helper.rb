require 'integration_helper'
require 'yaml'

class DiagnosticsTest < ActionDispatch::IntegrationTest

  def visit_page_with_token token_name, path='/'
    tokens = Rails.configuration.diagnostics_testing_user_tokens
    visit page_with_token(tokens[token_name], path)
  end

  def diagnostic_test_pipeline_config pipeline_to_run
    Rails.configuration.diagnostics_testing_pipeline_fields[pipeline_to_run]
  end

end
