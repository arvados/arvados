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

      # Run the pipeline
      find('a,button', text: 'Run').click

      # Choose project
      within('.modal-dialog') do
        find('.selectable', text: 'Home').click
        find('button', text: 'Choose').click
      end

      # Choose input for the pipeline
      if pipeline_config['input_paths'].andand.any?
        # This pipeline needs input. So, Run should be disabled
        page.assert_selector 'a.disabled,button.disabled', text: 'Run'

        inputs_needed = page.all('.btn', text: 'Choose')
        inputs_needed.each_with_index do |input_needed, index|
          input_needed.click
          within('.modal-dialog') do
            look_for = pipeline_config['input_paths'][index]
            found = page.has_text?(look_for)
            if found
              find('.selectable').click
            else
              fill_in('Search', with: look_for, exact: true)
              wait_for_ajax
              find('.selectable').click
            end
            find('button', text: 'OK').click
            wait_for_ajax
          end
        end
      end

      # Run this pipeline instance
      find('a,button', text: 'Run').click

      # Pipeline is running. We have a "Stop" button instead now.
      page.assert_selector 'a,button', text: 'Stop'
    end
  end

end
