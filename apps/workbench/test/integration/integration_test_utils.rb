# This file is used to define methods reusable by two or more integration tests
#

# check_checkboxes_state asserts that the page holds at least one
# checkbox matching 'selector', and that all matching checkboxes
# are in state 'checkbox_status' (i.e. checked if true, unchecked otherwise)
def assert_checkboxes_state(selector, checkbox_status, msg=nil)
  assert page.has_selector?(selector)
  page.all(selector).each do |checkbox|
    assert(checkbox.checked? == checkbox_status, msg)
  end
end
