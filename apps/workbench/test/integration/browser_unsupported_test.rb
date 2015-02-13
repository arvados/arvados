require 'integration_helper'

class BrowserUnsupported < ActionDispatch::IntegrationTest
  WARNING_FRAGMENT = 'Your web browser is missing some of the features'

  test 'warning if no File API' do
    Capybara.current_driver = :poltergeist_without_file_api
    visit '/'
    assert_text :visible, WARNING_FRAGMENT
  end

  test 'no warning if File API' do
    need_javascript
    visit '/'
    assert_no_text :visible, WARNING_FRAGMENT
  end
end
