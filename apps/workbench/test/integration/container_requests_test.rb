require 'integration_helper'

class ContainerRequestsTest < ActionDispatch::IntegrationTest
  setup do
    need_javascript
  end

  test "enter a float for a number workflow input" do
    # Poltergeist either does not support the HTML 5 <input
    # type="number">, or interferes with the associated X-Editable
    # validation code.  If the input field has type=number (forcing an
    # integer), this test will yield a false positive under
    # Poltergeist.  --Brett, 2015-02-05
    need_selenium "for strict X-Editable input validation"
    request_uuid = api_fixture("container_requests", "uncommitted", "uuid")
    visit page_with_token("active", "/container_requests/#{request_uuid}")
    INPUT_SELECTOR =
      ".editable[data-name='[mounts][/var/lib/cwl/cwl.input.json][content][ex_double]']"
    find(INPUT_SELECTOR).click
    find(".editable-input input").set("12.34")
    find("#editable-submit").click
    assert_no_selector(".editable-popup")
    assert_selector(INPUT_SELECTOR, text: "12.34")
  end

end
