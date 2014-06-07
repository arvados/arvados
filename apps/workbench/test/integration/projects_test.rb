require 'integration_helper'
require 'selenium-webdriver'
require 'headless'

class ProjectsTest < ActionDispatch::IntegrationTest
  setup do
    Capybara.current_driver = Capybara.javascript_driver
  end

  test 'Find a project and edit its description' do
    visit page_with_token 'active', '/'
    find('.arv-project-list a,button', text: 'A Project').
      click
    within('.container-fluid', text: api_fixture('groups')['aproject']['name']) do
      find('span', text: api_fixture('groups')['aproject']['name']).click
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
    project_uuid = api_fixture('groups')['aproject']['uuid']
    specimen_uuid = api_fixture('specimens')['owned_by_aproject_with_no_name_link']['uuid']
    visit page_with_token 'active', '/projects/' + project_uuid
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

  test 'Create a project and move it into a different project' do
    visit page_with_token 'active', '/projects'
    find('.btn', text: "Add new project").click

    # within('.editable', text: 'New project') do
    within('h2') do
      find('.fa-pencil').click
      find('.editable-input input').set('Project 1234')
      find('.glyphicon-ok').click
    end
    wait_for_ajax

    visit '/projects'
    find('.btn', text: "Add new project").click
    within('h2') do
      find('.fa-pencil').click
      find('.editable-input input').set('Project 5678')
      find('.glyphicon-ok').click
    end
    wait_for_ajax

    click_link 'Permissions'
    find('input[value="Move to..."]').click
    find('.selectable', text: 'Project 1234').click
    find('a,button', text: 'Move').click
    wait_for_ajax

    # Wait for the page to refresh and show the new parent in Permissions panel
    click_link 'Permissions'
    find('.panel', text: 'Project 1234')

    assert(find('.panel', text: 'Permissions for this project are inherited by the owner or parent project').
           all('*', text: 'Project 1234').any?,
           "Project 5678 should now be inside project 1234")
  end

end
