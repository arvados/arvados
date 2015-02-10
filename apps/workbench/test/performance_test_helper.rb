require 'integration_helper'

# Performance test can run in two different ways:
#
# 1. Similar to other integration tests using the command:
#     RAILS_ENV=test bundle exec rake test:benchmark
#
# 2. Against a configured workbench url using "RAILS_ENV=performance".
#     RAILS_ENV=performance bundle exec rake test:benchmark

class WorkbenchPerformanceTest < ActionDispatch::PerformanceTest

  # When running in "RAILS_ENV=performance" mode, uses performance
  # config params.  In this mode, prepends workbench URL to the given
  # path provided, and visits that page using the configured
  # "user_token".
  def visit_page_with_token path='/'
    if Rails.env == 'performance'
      token = Rails.configuration.user_token
      workbench_url = Rails.configuration.arvados_workbench_url
      if workbench_url.end_with? '/'
        workbench_url = workbench_url[0, workbench_url.size-1]
      end
    else
      token = 'active'
      workbench_url = ''
    end

    visit page_with_token(token, (workbench_url + path))
  end

end
