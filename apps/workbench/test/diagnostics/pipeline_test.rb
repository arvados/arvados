require 'diagnostics_test_helper'
require 'selenium-webdriver'
require 'headless'

class PipelineTest < DiagnosticsTest
  reset_api_fixtures :after_each_test, false
  reset_api_fixtures :after_suite, false
  reset_api_fixtures :before_suite, false

  pipelines_to_test = Rails.configuration.pipelines_to_test.andand.keys

  setup do
    headless = Headless.new
    headless.start
    Capybara.current_driver = :selenium
  end

  pipelines_to_test.andand.each do |pipeline_to_test|
    test "visit home page for user #{pipeline_to_test}" do
      visit_page_with_token 'active'
      pipeline_config = Rails.configuration.pipelines_to_test[pipeline_to_test]

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

      page.assert_selector('a.disabled,button.disabled', text: 'Run') if pipeline_config['input_paths'].any?

      # Choose input for the pipeline
      pipeline_config['input_paths'].each do |look_for|
        select_input look_for
      end
      wait_for_ajax

      # All needed input are filled in. Run this pipeline now
      find('a,button', text: 'Run').click

      # Pipeline is running. We have a "Stop" button instead now.
      page.assert_selector 'a,button', text: 'Pause'

      # Wait for pipeline run to complete
      wait_until_page_has 'Complete', pipeline_config['max_wait_seconds']
    end
  end

  def select_input look_for
    inputs_needed = page.all('.btn', text: 'Choose')
    return if (!inputs_needed || !inputs_needed.any?)

    look_for_uuid = nil
    look_for_file = nil
    if look_for.andand.index('/').andand.>0
      partitions = look_for.partition('/')
      look_for_uuid = partitions[0]
      look_for_file = partitions[2]
    else
      look_for_uuid = look_for
      look_for_file = nil
    end

    inputs_needed[0].click

    within('.modal-dialog') do
      if look_for_uuid
        fill_in('Search', with: look_for_uuid, exact: true)
        wait_for_ajax
      end
             
      page.all('.selectable').first.click
      wait_for_ajax
      # ajax reload is wiping out input selection after search results; so, select again.
      page.all('.selectable').first.click
      wait_for_ajax

      if look_for_file
        wait_for_ajax
        within('.collection_files_name', text: look_for_file) do
          find('.fa-file').click
        end
      end
      
      find('button', text: 'OK').click
      wait_for_ajax
    end
  end
end
