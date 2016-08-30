require 'integration_helper'

class WorkUnitsTest < ActionDispatch::IntegrationTest
  setup do
    need_javascript
  end

  test "scroll all_processes page" do
      expected_min, expected_max, expected, not_expected = [
        25, 100,
        ['/pipeline_instances/zzzzz-d1hrv-1yfj61234abcdk3',
         '/pipeline_instances/zzzzz-d1hrv-jobspeccomponts',
         '/jobs/zzzzz-8i9sb-grx15v5mjnsyxk7',
         '/jobs/zzzzz-8i9sb-n7omg50bvt0m1nf',
         '/container_requests/zzzzz-xvhdp-cr4completedcr2',
         '/container_requests/zzzzz-xvhdp-cr4requestercn2'],
        ['/pipeline_instances/zzzzz-d1hrv-scarxiyajtshq3l',
         '/container_requests/zzzzz-xvhdp-oneof60crs00001']
      ]

      visit page_with_token('active', "/all_processes")

      page_scrolls = expected_max/20 + 2
      within('.arv-recent-all-processes') do
        (0..page_scrolls).each do |i|
          page.driver.scroll_to 0, 999000
          begin
            wait_for_ajax
          rescue
          end
        end
      end

      # Verify that expected number of processes are found
      found_items = page.all('tr[data-object-uuid]')
      found_count = found_items.count
      if expected_min == expected_max
        assert_equal(true, found_count == expected_min,
          "Not found expected number of items. Expected #{expected_min} and found #{found_count}")
        assert page.has_no_text? 'request failed'
      else
        assert_equal(true, found_count>=expected_min,
          "Found too few items. Expected at least #{expected_min} and found #{found_count}")
        assert_equal(true, found_count<=expected_max,
          "Found too many items. Expected at most #{expected_max} and found #{found_count}")
      end

      # verify that all expected uuid links are found
      expected.each do |link|
        assert_selector "a[href=\"#{link}\"]"
      end

      # verify that none of the not_expected uuid links are found
      not_expected.each do |link|
        assert_no_selector "a[href=\"#{link}\"]"
      end
  end

  [
    ['jobs', 'running_job_with_components', true],
    ['pipeline_instances', 'components_is_jobspec', false],
    ['containers', 'running', false],
    ['container_requests', 'running', true],
  ].each do |type, fixture, cancelable|
    test "cancel button for #{type}/#{fixture}" do
      if cancelable
        need_selenium 'to cancel'
      end

      obj = api_fixture(type)[fixture]
      visit page_with_token "active", "/#{type}/#{obj['uuid']}"

      assert_text 'created_at'
      if cancelable
        assert_selector 'button', text: 'Cancel'
        click_button 'Cancel'
        wait_for_ajax
      end
      assert_no_selector 'button', text: 'Cancel'
    end
  end

  [
    ['jobs', 'running_job_with_components'],
    ['pipeline_instances', 'has_component_with_completed_jobs'],
    ['container_requests', 'running'],
    ['container_requests', 'completed'],
  ].each do |type, fixture|
    test "edit description for #{type}/#{fixture}" do
      obj = api_fixture(type)[fixture]
      visit page_with_token "active", "/#{type}/#{obj['uuid']}"

      within('.arv-description-as-subtitle') do
        find('.fa-pencil').click
        find('.editable-input textarea').set('*Textile description for object*')
        find('.editable-submit').click
      end
      wait_for_ajax

      # verify description
      assert page.has_no_text? '*Textile description for object*'
      assert page.has_text? 'Textile description for object'
    end
  end

  [
    ['Two Part Pipeline Template', 'part-one', 'Provide a value for the following'],
    ['Workflow with input specifications', 'this workflow has inputs specified', 'Provide a value for the following'],
  ].each do |template_name, preview_txt, process_txt|
    test "run a process using template #{template_name} from dashboard" do
      visit page_with_token('admin')
      assert_text 'Recent pipelines and processes' # seeing dashboard now

      within('.recent-processes-actions') do
        assert page.has_link?('All processes')
        find('a', text: 'Run a pipeline').click
      end

      # in the chooser, verify preview and click Next button
      within('.modal-dialog') do
        find('.selectable', text: template_name).click
        assert_text preview_txt
        find('.btn', text: 'Next: choose inputs').click
      end

      # in the process page now
      assert_text process_txt
    end
  end
end
