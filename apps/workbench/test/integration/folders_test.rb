require 'integration_helper'
require 'selenium-webdriver'
require 'headless'

class FoldersTest < ActionDispatch::IntegrationTest
  setup do
    Capybara.current_driver = Capybara.javascript_driver
  end

  test 'Find a folder and edit its description' do
    visit page_with_token 'active', '/'
    find('.arv-folder-list a,button', text: 'A Folder').
      click
    within('.container-fluid', text: api_fixture('groups')['afolder']['name']) do
      find('span', text: api_fixture('groups')['afolder']['name']).click
      within('.arv-description-as-subtitle') do
        find('.fa-pencil').click
        find('.editable-input textarea').set('I just edited this.')
        find('.editable-submit').click
      end
      wait_for_ajax
    end
    visit current_path
    assert(find?('.container-fluid', text: 'I just edited this.'),
           "Description update did not survive page refresh")
  end

  test 'Add a new name, then edit it, without creating a duplicate' do
    folder_uuid = api_fixture('groups')['afolder']['uuid']
    specimen_uuid = api_fixture('specimens')['owned_by_afolder_with_no_name_link']['uuid']
    visit page_with_token 'active', '/folders/' + folder_uuid
    within(".selection-action-container") do
      within (first('tr', text: 'Specimen')) do
        find(".fa-pencil").click
        find('.editable-input input').set('Now I have a name.')
        find('.glyphicon-ok').click
        find('.editable', text: 'Now I have a name.').click
        find(".fa-pencil").click
        find('.editable-input input').set('Now I have a new name.')
        find('.glyphicon-ok').click
        end
      wait_for_ajax
      find('.editable', text: 'Now I have a new name.')
    end
    visit current_path
    within '.selection-action-container' do
      find '.editable', text: 'Now I have a new name.'
      page.assert_no_selector '.editable', text: 'Now I have a name.'
    end
  end

  test 'Create a folder and move it into a different folder' do
    visit page_with_token 'active', '/folders'
    find('.btn', text: "Add new folder").click

    # within('.editable', text: 'New folder') do
    within('h2') do
      find('.fa-pencil').click
      find('.editable-input input').set('Folder 1234')
      find('.glyphicon-ok').click
    end
    wait_for_ajax

    visit '/folders'
    find('.btn', text: "Add new folder").click
    within('h2') do
      find('.fa-pencil').click
      find('.editable-input input').set('Folder 5678')
      find('.glyphicon-ok').click
    end
    wait_for_ajax

    click_link 'Permissions'
    find('input[value="Move to..."]').click
    find('.selectable', text: 'Folder 1234').click
    find('a,button', text: 'Move').click
    wait_for_ajax

    # Wait for the page to refresh and show the new parent in Permissions panel
    click_link 'Permissions'
    find('.panel', text: 'Folder 1234')

    assert(find('.panel', text: 'Permissions for this folder are inherited by the owner or parent folder').
           all('*', text: 'Folder 1234').any?,
           "Folder 5678 should now be inside folder 1234")
  end

end
