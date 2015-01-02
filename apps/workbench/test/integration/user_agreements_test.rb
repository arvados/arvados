require 'integration_helper'

class UserAgreementsTest < ActionDispatch::IntegrationTest

  setup do
    Capybara.current_driver = Capybara.javascript_driver
  end

  def continuebutton_selector
    'input[type=submit][disabled][value=Continue]'
  end

  test "cannot click continue without ticking checkbox" do
    visit page_with_token('inactive')
    assert_selector continuebutton_selector
  end

  test "continue button is enabled after ticking checkbox" do
    visit page_with_token('inactive')
    assert_selector continuebutton_selector
    find('input[type=checkbox]').click
    assert_no_selector continuebutton_selector
    assert_nil(find_button('Continue')[:disabled],
               'Continue button did not become enabled')
  end

end
