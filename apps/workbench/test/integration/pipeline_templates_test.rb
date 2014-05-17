require 'integration_helper'
require 'selenium-webdriver'
require 'headless'

class PipelineTemplatesTest < ActionDispatch::IntegrationTest

  test 'Create new pipeline instance from template' do
    Capybara.current_driver = Capybara.javascript_driver
    uuid = api_fixture('pipeline_templates')['two_part']['uuid']
    visit page_with_token('active', '/pipeline_templates')
    within 'tr', text: uuid do
      find('button[type=submit]').click
      wait_for_ajax
    end
    page.assert_selector 'tr', text: 'part-one'
    page.assert_selector 'tr', text: 'part-two'
    page.assert_selector 'a,button', text: 'Run pipeline'
  end

end
