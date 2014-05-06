require 'integration_helper'
require 'selenium-webdriver'
require 'headless'

class FoldersTest < ActionDispatch::IntegrationTest

  test 'Find a folder and edit its description' do
    Capybara.current_driver = Capybara.javascript_driver
    visit page_with_token 'active', '/'
    find('nav a', text: 'Folders').click
    find('.side-nav a,button', text: 'A Folder').
      click
    within('.panel', text: api_fixture('groups')['afolder']['name']) do
      find('span', text: api_fixture('groups')['afolder']['name']).click
      find('.glyphicon-ok').click
      find('.btn', text: 'Edit description').click
      find('.editable-input textarea').set('I just edited this.')
      find('.editable-submit').click
      # Wait for editable popup to go away
      page.assert_no_selector '.editable-submit'
    end
    visit current_path
    assert(find?('.panel', text: 'I just edited this.'),
           "Description update did not survive page refresh")
  end

  test 'Add a new name, then edit it, without creating a duplicate' do
    Capybara.current_driver = Capybara.javascript_driver
    folder_uuid = api_fixture('groups')['afolder']['uuid']
    specimen_uuid = api_fixture('specimens')['owned_by_afolder_with_no_name_link']['uuid']
    visit page_with_token 'active', '/folders/' + folder_uuid
    within('.panel tr', text: specimen_uuid) do
      find(".editable[data-name='name']").click
      find('.editable-input input').set('Now I have a name.')
      find('.glyphicon-ok').click
      find('.editable', text: 'Now I have a name.').click
      find('.editable-input input').set('Now I have a new name.')
      find('.glyphicon-ok').click
      find('.editable', text: 'Now I have a new name.')
    end
    visit current_path
    within '.panel', text: 'Contents' do
      find '.editable', text: 'Now I have a new name.'
      page.assert_no_selector '.editable', text: 'Now I have a name.'
    end
  end

end
