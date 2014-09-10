require 'diagnostics_test_helper'
require 'selenium-webdriver'
require 'headless'

class PipelineTest < DiagnosticsTest
  pipelines_to_run = Rails.configuration.diagnostics_testing_pipeline_fields.andand.keys

  setup do
    headless = Headless.new
    headless.start
    Capybara.current_driver = :selenium
  end

  pipelines_to_run.andand.each do |pipeline_to_run|
    test "visit home page for user #{pipeline_to_run}" do
      visit_page_with_token 'active'

      pipeline_config = diagnostic_test_pipeline_config pipeline_to_run

      # Search for tutorial template
      within('.navbar-fixed-top') do
        page.find_field('search').set pipeline_config['template_uuid']
        page.find('.glyphicon-search').click
      end

#      within '.modal-content' do
#        find('.selectable', text: pipeline_config['template_name']).click
#        find(:xpath, "//div[./span[contains(.,'zzzzz-p5p6p-rxj8d71854j9idn')]]").click
#        click_button 'Show'
#      end

      # Run the pipeline
      find('a,button', text: 'Run').click

      # Choose project
      within('.modal-dialog') do
        find('.selectable', text: 'Home').click
        find('button', text: 'Choose').click
      end

        # This pipeline needs input. So, Run should be disabled
        page.assert_selector 'a.disabled,button.disabled', text: 'Run'

        # Choose input for the pipeline
        find('.btn', text: 'Choose').click
        within('.modal-dialog') do
          find('.selectable', text: pipeline_config['input_names'][0]).click
          find('button', text: 'OK').click
        end
        wait_for_ajax

        # Run this pipeline instance
        find('a,button', text: 'Run').click

        # Pipeline is running. We have a "Stop" button instead now.
        page.assert_selector 'a,button', text: 'Stop'
      end
  end

end
