require 'helpers/fake_websocket_helper'
require 'integration_helper'

class WorkUnitsTest < ActionDispatch::IntegrationTest
  include FakeWebsocketHelper

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
        assert_text 'priority: 1' if type.include?('container')
        assert_selector 'button', text: 'Cancel'
        first('a,button', text: 'Cancel').click
        wait_for_ajax
      end
      assert_text 'priority: 0' if cancelable and type.include?('container')
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
      assert_selector 'a', text: template_name
    end
  end

  test 'display container state changes in Container Request live log' do
    use_fake_websocket_driver
    c = api_fixture('containers')['queued']
    cr = api_fixture('container_requests')['queued']
    visit page_with_token('active', '/container_requests/'+cr['uuid'])
    click_link('Log')

    # The attrs of the "terminal window" text div in the log tab
    # indicates which objects' events are worth displaying. Events
    # that arrive too early (before that div exists) are not
    # shown. For the user's sake, these early logs should also be
    # retrieved and shown one way or another -- but in this particular
    # test, we are only interested in logs that arrive by
    # websocket. Therefore, to avoid races, we wait for the log tab to
    # display before sending any events.
    assert_text 'Recent logs'

    [[{
        event_type: 'dispatch',
        properties: {
          text: "dispatch logged a fake message\n",
        },
      }, "dispatch logged"],
     [{
        event_type: 'update',
        properties: {
          old_attributes: {state: 'Locked'},
          new_attributes: {state: 'Queued'},
        },
      }, "Container #{c['uuid']} was returned to the queue"],
     [{
        event_type: 'update',
        properties: {
          old_attributes: {state: 'Queued'},
          new_attributes: {state: 'Locked'},
        },
      }, "Container #{c['uuid']} was taken from the queue by a dispatch process"],
     [{
        event_type: 'crunch-run',
        properties: {
          text: "according to fake crunch-run,\nsome setup stuff happened on the compute node\n",
        },
      }, "setup stuff happened"],
     [{
        event_type: 'update',
        properties: {
          old_attributes: {state: 'Locked'},
          new_attributes: {state: 'Running'},
        },
      }, "Container #{c['uuid']} started"],
     [{
        event_type: 'update',
        properties: {
          old_attributes: {state: 'Running'},
          new_attributes: {state: 'Complete', exit_code: 1},
        },
      }, "Container #{c['uuid']} finished with exit code 1 (failure)"],
     # It's unrealistic for state to change again once it's Complete,
     # but the logging code doesn't care, so we do it to keep the test
     # simple.
     [{
        event_type: 'update',
        properties: {
          old_attributes: {state: 'Running'},
          new_attributes: {state: 'Cancelled'},
        },
      }, "Container #{c['uuid']} was cancelled"],
    ].each do |send_event, expect_log_text|
      assert_no_text(expect_log_text)
      fake_websocket_event(send_event.merge(object_uuid: c['uuid']))
      assert_text(expect_log_text)
    end
  end

  [
    ['jobs', 'active', 'running_job_with_components', 'component1', '/jobs/zzzzz-8i9sb-jyq01m7in1jlofj#Log'],
    ['pipeline_instances', 'active', 'pipeline_in_running_state', 'foo', '/jobs/zzzzz-8i9sb-pshmckwoma9plh7#Log'],
    ['pipeline_instances', nil, 'pipeline_in_publicly_accessible_project_but_other_objects_elsewhere', 'foo', 'Log unavailable'],
  ].each do |type, token, fixture, child, log_link|
    test "link_to_log for #{fixture} for #{token}" do
      obj = api_fixture(type)[fixture]
      if token
        visit page_with_token token, "/#{type}/#{obj['uuid']}"
      else
        Rails.configuration.anonymous_user_token =
          api_fixture("api_client_authorizations", "anonymous", "api_token")
        visit "/#{type}/#{obj['uuid']}"
      end

      click_link(child)

      if token
        assert_selector "a[href=\"#{log_link}\"]"
      else
        assert_text log_link
      end
    end
  end
end
