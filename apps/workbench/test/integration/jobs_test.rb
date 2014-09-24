require 'integration_helper'

class JobsTest < ActionDispatch::IntegrationTest
  test "add job description" do
    Capybara.current_driver = Capybara.javascript_driver
    visit page_with_token("active", "/jobs")

    # go to job running the script "doesnotexist"
    within first('tr', text: 'doesnotexist') do
      find("a").click
    end

    # edit job description
    within('.arv-description-as-subtitle') do
      find('.fa-pencil').click
      find('.editable-input textarea').set('*Textile description for job* - "Go to dashboard":/')
      find('.editable-submit').click
    end
    wait_for_ajax

    # Verify edited description
    assert page.has_no_text? '*Textile description for job*'
    assert page.has_text? 'Textile description for job'
    assert page.has_link? 'Go to dashboard'
    click_link 'Go to dashboard'
    assert page.has_text? 'Active pipelines'
  end
end
