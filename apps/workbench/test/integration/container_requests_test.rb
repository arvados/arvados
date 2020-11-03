# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

require 'integration_helper'

class ContainerRequestsTest < ActionDispatch::IntegrationTest
  setup do
    need_javascript
  end

  [
    ['ex_string', 'abc'],
    ['ex_string_opt', 'abc'],
    ['ex_int', 12],
    ['ex_int_opt', 12],
    ['ex_long', 12],
    ['ex_double', '12.34', 12.34],
    ['ex_float', '12.34', 12.34],
  ].each do |input_id, input_value, expected_value|
    test "set input #{input_id} with #{input_value}" do
      request_uuid = api_fixture("container_requests", "uncommitted", "uuid")
      visit page_with_token("active", "/container_requests/#{request_uuid}")
      selector = ".editable[data-name='[mounts][/var/lib/cwl/cwl.input.json][content][#{input_id}]']"
      find(selector).click
      find(".editable-input input").set(input_value)
      find("#editable-submit").click
      assert_no_selector(".editable-popup")
      assert_selector(selector, text: expected_value || input_value)
    end
  end

  test "select value for boolean input" do
    request_uuid = api_fixture("container_requests", "uncommitted", "uuid")
    visit page_with_token("active", "/container_requests/#{request_uuid}")
    selector = ".editable[data-name='[mounts][/var/lib/cwl/cwl.input.json][content][ex_boolean]']"
    find(selector).click
    within(".editable-input") do
      select "true"
    end
    find("#editable-submit").click
    assert_no_selector(".editable-popup")
    assert_selector(selector, text: "true")
  end

  test "select value for enum typed input" do
    request_uuid = api_fixture("container_requests", "uncommitted", "uuid")
    visit page_with_token("active", "/container_requests/#{request_uuid}")
    selector = ".editable[data-name='[mounts][/var/lib/cwl/cwl.input.json][content][ex_enum]']"
    find(selector).click
    within(".editable-input") do
      select "b"    # second value
    end
    find("#editable-submit").click
    assert_no_selector(".editable-popup")
    assert_selector(selector, text: "b")
  end

  [
    ['directory_type'],
    ['file_type'],
  ].each do |type|
    test "select value for #{type} input" do
      request_uuid = api_fixture("container_requests", "uncommitted-with-directory-input", "uuid")
      visit page_with_token("active", "/container_requests/#{request_uuid}")
      assert_text 'Provide a value for the following parameter'
      click_link 'Choose'
      within('.modal-dialog') do
        wait_for_ajax
        collection = api_fixture('collections', 'collection_with_one_property', 'uuid')
        find("div[data-object-uuid=#{collection}]").click
        if type == 'ex_file'
          wait_for_ajax
          find('.preview-selectable', text: 'bar').click
        end
        find('button', text: 'OK').click
      end
      page.assert_no_selector 'a.disabled,button.disabled', text: 'Run'
      assert_text 'This workflow does not need any further inputs'
      click_link "Run"
      wait_for_ajax
      assert_text 'This container is queued'
    end
  end

  test "Run button enabled once all required inputs are provided" do
    request_uuid = api_fixture("container_requests", "uncommitted-with-required-and-optional-inputs", "uuid")
    visit page_with_token("active", "/container_requests/#{request_uuid}")
    assert_text 'Provide a value for the following parameter'

    page.assert_selector 'a.disabled,button.disabled', text: 'Run'

    selector = ".editable[data-name='[mounts][/var/lib/cwl/cwl.input.json][content][int_required]']"
    find(selector).click
    find(".editable-input input").set(2016)
    find("#editable-submit").click

    page.assert_no_selector 'a.disabled,button.disabled', text: 'Run'
    click_link "Run"
    wait_for_ajax
    assert_text 'This container is queued'
  end

  test "Run button enabled when workflow is empty and no inputs are needed" do
    visit page_with_token("active")

    find('.btn', text: 'Run a process').click
    within('.modal-dialog') do
      find('.selectable', text: 'Valid workflow with no definition yaml').click
      find('.btn', text: 'Next: choose inputs').click
    end

    assert_text 'This workflow does not need any further inputs'
    page.assert_selector 'a', text: 'Run'
  end

  test "Provenance graph shown on committed container requests" do
    cr = api_fixture('container_requests', 'completed')
    visit page_with_token("active", "/container_requests/#{cr['uuid']}")
    assert page.has_text? 'Provenance'
    click_link 'Provenance'
    wait_for_ajax
    # Check for provenance graph existance
    page.assert_selector '#provenance_svg'
    page.assert_selector 'ellipse+text', text: cr['name'], visible: false
    page.assert_selector 'g.node>title', text: cr['uuid'], visible: false
  end

  test "index page" do
    visit page_with_token("active", "/container_requests")

    within(".arv-recent-container-requests") do
      page.execute_script "window.scrollBy(0,999000)"
      wait_for_ajax
    end

    running_owner_active = api_fixture("container_requests", "requester_for_running")
    anon_accessible_cr = api_fixture("container_requests", "running_anonymous_accessible")

    # both of these CRs should be accessible to the user
    assert_selector "a[href=\"/container_requests/#{running_owner_active['uuid']}\"]", text: running_owner_active[:name]
    assert_selector "a[href=\"/container_requests/#{anon_accessible_cr['uuid']}\"]", text: anon_accessible_cr[:name]

    # user can delete the "running" container_request
    within(".cr-#{running_owner_active['uuid']}") do
      assert_not_nil first('.glyphicon-trash')
    end

    # user can not delete the anonymously accessible container_request
    within(".cr-#{anon_accessible_cr['uuid']}") do
      assert_nil first('.glyphicon-trash')
    end

    # verify the search box in the page
    find('.recent-container-requests-filterable-control').set("anonymous")
    sleep 0.350 # Wait for 250ms debounce timer (see filterable.js)
    wait_for_ajax
    assert_no_selector "a[href=\"/container_requests/#{running_owner_active['uuid']}\"]", text: running_owner_active[:name]
    assert_selector "a[href=\"/container_requests/#{anon_accessible_cr['uuid']}\"]", text: anon_accessible_cr[:name]
  end
end
