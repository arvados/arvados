require 'integration_helper'

class PipelineTemplatesTest < ActionDispatch::IntegrationTest
  test "JSON popup available for strange components" do
    need_javascript
    uuid = api_fixture("pipeline_templates")["components_is_jobspec"]["uuid"]
    visit page_with_token("active", "/pipeline_templates/#{uuid}")
    click_on "Components"
    assert(page.has_no_text?("script_parameters"),
           "components JSON visible without popup")
    click_on "Show components JSON"
    assert(page.has_text?("script_parameters"),
           "components JSON not found")
  end

  test "pipeline template description" do
    need_javascript
    visit page_with_token("active", "/pipeline_templates")

    # go to Two Part pipeline template
    within first('tr', text: 'Two Part Pipeline Template') do
      find(".fa-gears").click
    end

    # edit template description
    within('.arv-description-as-subtitle') do
      find('.fa-pencil').click
      find('.editable-input textarea').set('*Textile description for pipeline template* - "Go to dashboard":/')
      find('.editable-submit').click
    end
    wait_for_ajax

    # Verfiy edited description
    assert page.has_no_text? '*Textile description for pipeline template*'
    assert page.has_text? 'Textile description for pipeline template'
    assert page.has_link? 'Go to dashboard'
    click_link 'Go to dashboard'
    assert page.has_text? 'Active pipelines'

    # again visit recent templates page and verify edited description
    visit page_with_token("active", "/pipeline_templates")
    assert page.has_no_text? '*Textile description for pipeline template*'
    assert page.has_text? 'Textile description for pipeline template'
  end
end
