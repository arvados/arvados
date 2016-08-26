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
    ['ex_double', 12.34],
    ['ex_double', 'abc', true, 0.0],
    ['ex_float', 12.34],
    ['ex_float', 'abc', true, 0.0],
  ].each do |input_id, input_value, need_refresh, expected|
    test "set input #{input_id} with #{input_value}" do
      request_uuid = api_fixture("container_requests", "uncommitted", "uuid")
      visit page_with_token("active", "/container_requests/#{request_uuid}")
      selector = ".editable[data-name='[mounts][/var/lib/cwl/cwl.input.json][content][#{input_id}]']"
      find(selector).click
      find(".editable-input input").set(input_value)
      find("#editable-submit").click
      assert_no_selector(".editable-popup")
      assert_selector(selector, text: input_value)
      if need_refresh
        visit page_with_token("active", "/container_requests/#{request_uuid}")
        assert_selector(selector, text: expected)
      end
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
    test "select input for #{type}" do
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
      assert_text 'This container is committed'
    end
  end

  test "Run button enabled once all required inputs are provided" do
    request_uuid = api_fixture("container_requests", "uncommitted-with-required-and-optional-inputs", "uuid")
    visit page_with_token("active", "/container_requests/#{request_uuid}")
    assert_text 'Provide a value for the following parameter'

    selector = ".editable[data-name='[mounts][/var/lib/cwl/cwl.input.json][content][int_required]']"
    find(selector).click
    find(".editable-input input").set(2016)
    find("#editable-submit").click

    page.assert_no_selector 'a.disabled,button.disabled', text: 'Run'
    click_link "Run"
    wait_for_ajax
    assert_text 'This container is committed'
  end
end
