# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

require 'helpers/fake_websocket_helper'
require 'integration_helper'

class WorkUnitsTest < ActionDispatch::IntegrationTest
  include FakeWebsocketHelper

  setup do
    need_javascript
  end

  [[true, 25, 100,
    ['/pipeline_instances/zzzzz-d1hrv-1yfj61234abcdk3',
     '/pipeline_instances/zzzzz-d1hrv-1yfj61234abcdk4',
     '/jobs/zzzzz-8i9sb-grx15v5mjnsyxk7',
     '/jobs/zzzzz-8i9sb-n7omg50bvt0m1nf',
     '/container_requests/zzzzz-xvhdp-cr4completedcr2',
     '/container_requests/zzzzz-xvhdp-cr4requestercn2'],
    ['/pipeline_instances/zzzzz-d1hrv-scarxiyajtshq3l',
     '/container_requests/zzzzz-xvhdp-oneof60crs00001']],
   [false, 25, 100,
    ['/pipeline_instances/zzzzz-d1hrv-1yfj61234abcdk3',
     '/pipeline_instances/zzzzz-d1hrv-1yfj61234abcdk4',
     '/container_requests/zzzzz-xvhdp-cr4completedcr2'],
    ['/pipeline_instances/zzzzz-d1hrv-scarxiyajtshq3l',
     '/container_requests/zzzzz-xvhdp-oneof60crs00001',
     '/jobs/zzzzz-8i9sb-grx15v5mjnsyxk7',
     '/jobs/zzzzz-8i9sb-n7omg50bvt0m1nf',
     '/container_requests/zzzzz-xvhdp-cr4requestercn2'
    ]]
  ].each do |show_children, expected_min, expected_max, expected, not_expected|
    test "scroll all_processes page with show_children=#{show_children}" do
      visit page_with_token('active', "/all_processes")

      if show_children
        find('#IncludeChildProcs').click
        wait_for_ajax
      end

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
  end

  [
    ['containers', 'running', false],
    ['container_requests', 'running', true],
  ].each do |type, fixture, cancelable, confirm_cancellation|
    test "cancel button for #{type}/#{fixture}" do
      if cancelable
        need_selenium 'to cancel'
      end

      obj = api_fixture(type)[fixture]
      visit page_with_token "active", "/#{type}/#{obj['uuid']}"

      assert_text 'created_at'
      if cancelable
        assert_text 'priority: 501' if type.include?('container')
        if type.include?('pipeline')
          assert_selector 'a', text: 'Pause'
          first('a,link', text: 'Pause').click
        else
          assert_selector 'button', text: 'Cancel'
          first('a,button', text: 'Cancel').click
        end
        if confirm_cancellation
          alert = page.driver.browser.switch_to.alert
          alert.accept
        end
        wait_for_ajax
      end

      if type.include?('pipeline')
        assert_selector 'a', text: 'Resume'
        assert_no_selector 'a', text: 'Pause'
      elsif type.include?('job')
        assert_text 'Cancelled'
        assert_text 'Paused'  # this job has a pipeline child which was also cancelled
        assert_no_selector 'button', text: 'Cancel'
      elsif cancelable
        assert_text 'priority: 0'
      end
    end
  end

  [
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
    ['Workflow with default input specifications', 'this workflow has inputs specified', 'Provide a value for the following'],
  ].each do |template_name, preview_txt, process_txt|
    test "run a process using template #{template_name} from dashboard" do
      visit page_with_token('admin')
      assert_text 'Recent processes' # seeing dashboard now

      within('.recent-processes-actions') do
        assert page.has_link?('All processes')
        find('a', text: 'Run a process').click
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

      assert_equal "true", find('span[data-name="reuse_steps"]').text

      assert_equal "Set value for ex_string_def", find('div.form-group > div.form-control-static > a', text: "hello-testing-123")[:"data-title"]

      page.assert_selector 'a.disabled,button.disabled', text: 'Run'
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
      }, "Container #{c['uuid']} finished"],
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

  test 'Run from workflows index page' do
    visit page_with_token('active', '/workflows')

    wf_count = page.all('a[data-original-title="show workflow"]').count
    assert_equal true, wf_count>0

    # Run one of the workflows
    wf_name = 'Workflow with input specifications'
    within('tr', text: wf_name) do
      find('a,button', text: 'Run').click
    end

    # Choose project for the container_request being created
    within('.modal-dialog') do
      find('.selectable', text: 'A Project').click
      find('button', text: 'Choose').click
    end

    # In newly created container_request page now
    assert_text 'A Project' # CR created in "A Project"
    assert_text "This container request was created from the workflow #{wf_name}"
    assert_match /Provide a value for .* then click the \"Run\" button to start the workflow/, page.text
  end

  test 'Run workflow from show page' do
    visit page_with_token('active', '/workflows/zzzzz-7fd4e-validwithinputs')

    find('a,button', text: 'Run this workflow').click

    # Choose project for the container_request being created
    within('.modal-dialog') do
      find('.selectable', text: 'A Project').click
      find('button', text: 'Choose').click
    end

    # In newly created container_request page now
    assert_text 'A Project' # CR created in "A Project"
    assert_text "This container request was created from the workflow"
    assert_match /Provide a value for .* then click the \"Run\" button to start the workflow/, page.text
  end

  test "create workflow with WorkflowRunnerResources" do
    visit page_with_token('active', '/workflows/zzzzz-7fd4e-validwithinput3')

    find('a,button', text: 'Run this workflow').click

    # Choose project for the container_request being created
    within('.modal-dialog') do
      find('.selectable', text: 'A Project').click
      find('button', text: 'Choose').click
    end
    click_link 'Advanced'
    click_link("API response")
    assert_text('"container_image": "arvados/jobs:2.0.4"')
    assert_text('"vcpus": 2')
    assert_text('"ram": 1293942784')
    assert_text('"--collection-cache-size=678"')

  end
end
