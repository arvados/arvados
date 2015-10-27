require 'integration_helper'

class VirtualMachinesTest < ActionDispatch::IntegrationTest
  test "make and name a new virtual machine" do
    need_javascript
    visit page_with_token('admin_trustedclient')
    find('#system-menu').click
    click_link 'Virtual machines'
    assert page.has_text? 'testvm.shell'
    click_on 'Add a new virtual machine'
    find('tr', text: 'hostname').
      find('a[data-original-title=edit]').click
    assert page.has_text? 'Edit hostname'
    fill_in 'editable-text', with: 'testname'
    click_button 'editable-submit'
    assert page.has_text? 'testname'
  end
end
