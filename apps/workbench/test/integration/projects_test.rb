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

  test 'Find a project and edit description to textile description' do
    visit page_with_token 'active', '/'
    find('.arv-project-list a,button', text: 'A Project').
      click
    within('.container-fluid', text: api_fixture('groups')['aproject']['name']) do
      find('span', text: api_fixture('groups')['aproject']['name']).click
      within('.arv-description-as-subtitle') do
        find('.fa-pencil').click
        find('.editable-input textarea').set('<p>*Textile description for A project* - "take me home":/ </p><p>And a new paragraph in description.</p>')
        find('.editable-submit').click
      end
      wait_for_ajax
    end

    # visit project page
    visit current_path
    assert(has_no_text?('.container-fluid', text: '*Textile description for A project*'),
           "Description is not rendered properly")
    assert(find?('.container-fluid', text: 'Textile description for A project'),
           "Description update did not survive page refresh")
    assert(find?('.container-fluid', text: 'And a new paragraph in description'),
           "Description did not contain the expected new paragraph")
    assert(page.has_link?("take me home"), "link not found in description")

    click_link 'take me home'

    # now in dashboard
    assert(page.has_text?('My projects'), 'My projects - not found on dashboard')
    assert(page.has_text?('Projects shared with me'), 'Projects shared with me - not found on dashboard')
    assert(page.has_text?('Textile description for A project'), "Project description not found")
    assert(page.has_no_text?('*Textile description for A project*'), "Project description is not rendered properly in dashboard")
    assert(page.has_no_text?('And a new paragraph in description'), "Project description is not truncated after first paragraph")
  end

  test 'Find a project and edit description to html description' do
    visit page_with_token 'active', '/'
    find('.arv-project-list a,button', text: 'A Project').
      click
    within('.container-fluid', text: api_fixture('groups')['aproject']['name']) do
      find('span', text: api_fixture('groups')['aproject']['name']).click
      within('.arv-description-as-subtitle') do
        find('.fa-pencil').click
        find('.editable-input textarea').set('<br>Textile description for A project</br> - <a href="/">take me home</a>')
        find('.editable-submit').click
      end
      wait_for_ajax
    end
    visit current_path
    assert(find?('.container-fluid', text: 'Textile description for A project'),
           "Description update did not survive page refresh")
    assert(!find?('.container-fluid', text: '<br>Textile description for A project</br>'),
           "Textile description is displayed with uninterpreted formatting characters")
    assert(page.has_link?("take me home"),"link not found in description")
    click_link 'take me home'
    assert page.has_text?('My projects')
    assert page.has_text?('Projects shared with me')
  end

  test 'Find a project and edit description to textile description with link to object' do
    visit page_with_token 'active', '/'
    find('.arv-project-list a,button', text: 'A Project').
      click
    within('.container-fluid', text: api_fixture('groups')['aproject']['name']) do
      find('span', text: api_fixture('groups')['aproject']['name']).click
      within('.arv-description-as-subtitle') do
        find('.fa-pencil').click
        find('.editable-input textarea').set('*Textile description for A project* - "go to sub-project":' + api_fixture('groups')['asubproject']['uuid'] + "'")
        find('.editable-submit').click
      end
      wait_for_ajax
    end
    visit current_path
    assert(find?('.container-fluid', text: 'Textile description for A project'),
           "Description update did not survive page refresh")
    assert(!find?('.container-fluid', text: '*Textile description for A project*'),
           "Textile description is displayed with uninterpreted formatting characters")
    assert(page.has_link?("go to sub-project"), "link not found in description")
    click_link 'go to sub-project'
    assert(page.has_text?(api_fixture('groups')['asubproject']['name']), 'sub-project name not found after clicking link')
  end

  test 'Add a new name, then edit it, without creating a duplicate' do
    project_uuid = api_fixture('groups')['aproject']['uuid']
    specimen_uuid = api_fixture('traits')['owned_by_aproject_with_no_name']['uuid']
    visit page_with_token 'active', '/projects/' + project_uuid
    click_link 'Other objects'
    within '.selection-action-container' do
      # Wait for the tab to load:
      assert_selector 'tr[data-kind="arvados#trait"]'
      within first('tr', text: 'Trait') do
        find(".fa-pencil").click
        find('.editable-input input').set('Now I have a name.')
        find('.glyphicon-ok').click
        assert_selector '.editable', text: 'Now I have a name.'
        find(".fa-pencil").click
        find('.editable-input input').set('Now I have a new name.')
        find('.glyphicon-ok').click
      end
      wait_for_ajax
      assert_selector '.editable', text: 'Now I have a new name.'
    end
    visit current_path
    click_link 'Other objects'
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

    click_link 'Move project...'
    find('.selectable', text: 'Project 1234').click
    find('.modal-footer a,button', text: 'Move').click
    wait_for_ajax

    # Wait for the page to refresh and show the new parent in Sharing panel
    click_link 'Sharing'
    assert(page.has_link?("Project 1234"),
           "Project 5678 should now be inside project 1234")
  end

  def show_project_using(auth_key, proj_key='aproject')
    project_uuid = api_fixture('groups')[proj_key]['uuid']
    visit(page_with_token(auth_key, "/projects/#{project_uuid}"))
    assert(page.has_text?("A Project"), "not on expected project page")
  end

  def share_rows
    find('#project_sharing').all('tr')
  end

  def add_share_and_check(share_type, name)
    assert(page.has_no_text?(name), "project is already shared with #{name}")
    start_share_count = share_rows.size
    click_on("Share with #{share_type}")
    within(".modal-container") do
      # Order is important here: we should find something that appears in the
      # modal before we make any assertions about what's not in the modal.
      # Otherwise, the not-included assertions might falsely pass because
      # the modal hasn't loaded yet.
      find(".selectable", text: name).click
      assert(has_no_selector?(".modal-dialog-preview-pane"),
             "preview pane available in sharing dialog")
      assert_raises(Capybara::ElementNotFound,
                    "Projects pulldown available from sharing dialog") do
        click_on "All projects"
      end
      click_on "Add"
    end
    using_wait_time(Capybara.default_wait_time * 3) do
      assert(page.has_link?(name),
             "new share was not added to sharing table")
      assert_equal(start_share_count + 1, share_rows.size,
                   "new share did not add row to sharing table")
    end
  end

  def modify_share_and_check(name)
    start_rows = share_rows
    link_row = start_rows.select { |row| row.has_text?(name) }
    assert_equal(1, link_row.size, "row with new permission not found")
    within(link_row.first) do
      click_on("Read")
      select("Write", from: "share_change_level")
      click_on("editable-submit")
      assert(has_link?("Write"),
             "failed to change access level on new share")
      click_on "Revoke"
    end
    using_wait_time(Capybara.default_wait_time * 3) do
      assert(page.has_no_text?(name),
             "new share row still exists after being revoked")
      assert_equal(start_rows.size - 1, share_rows.size,
                   "revoking share did not remove row from sharing table")
    end
  end

  test "project viewer can't see project sharing tab" do
    show_project_using("project_viewer")
    assert(page.has_no_link?("Sharing"),
           "read-only project user sees sharing tab")
  end

  test "project owner can manage sharing for another user" do
    add_user = api_fixture('users')['future_project_user']
    new_name = ["first_name", "last_name"].map { |k| add_user[k] }.join(" ")

    show_project_using("active")
    click_on "Sharing"
    add_share_and_check("users", new_name)
    modify_share_and_check(new_name)
  end

  test "project owner can manage sharing for another group" do
    new_name = api_fixture('groups')['future_project_viewing_group']['name']

    show_project_using("active")
    click_on "Sharing"
    add_share_and_check("groups", new_name)
    modify_share_and_check(new_name)
  end

  test "'share with group' listing does not offer projects" do
    show_project_using("active")
    click_on "Sharing"
    click_on "Share with groups"
    good_uuid = api_fixture("groups")["private"]["uuid"]
    assert(page.has_selector?(".selectable[data-object-uuid=\"#{good_uuid}\"]"),
           "'share with groups' listing missing owned user group")
    bad_uuid = api_fixture("groups")["asubproject"]["uuid"]
    assert(page.has_no_selector?(".selectable[data-object-uuid=\"#{bad_uuid}\"]"),
           "'share with groups' listing includes project")
  end

  [
    'Move',
    'Remove',
    'Copy',
  ].each do |action|
    test "selection #{action} for project" do
      src = api_fixture('groups')['aproject']
      dest = api_fixture('groups')['asubproject']
      my_collection = api_fixture('collections')['collection_to_move_around_in_aproject']

      perform_selection_action src, dest, my_collection, action

      case action
      when 'Copy'
        assert page.has_text?(my_collection['name']), 'Collection not found in src project after copy'
        visit page_with_token 'active', '/'
        find('.arv-project-list a,button', text: dest['name']).click
        assert page.has_text?(my_collection['name']), 'Collection not found in dest project after copy'

        # now remove it from destination project to restore to original state
        perform_selection_action dest, nil, my_collection, 'Remove'
      when 'Move'
        assert page.has_no_text?(my_collection['name']), 'Collection still found in src project after move'
        visit page_with_token 'active', '/'
        find('.arv-project-list a,button', text: dest['name']).click
        assert page.has_text?(my_collection['name']), 'Collection not found in dest project after move'

        # move it back to src project to restore to original state
        perform_selection_action dest, src, my_collection, action
      when 'Remove'
        assert page.has_no_text?(my_collection['name']), 'Collection still found in src project after remove'
        visit page_with_token 'active', '/'
        find('.arv-project-list a,button', text: 'Home').click
        assert page.has_text?(my_collection['name']), 'Collection not found in home project after remove'
      end
    end
  end

  def perform_selection_action src, dest, item, action
    visit page_with_token 'active', '/'
    find('.arv-project-list a,button', text: src['name']).click
    assert page.has_text?(item['name']), 'Collection not found in src project'

    within('tr', text: item['name']) do
      find('input[type=checkbox]').click
    end

    click_button 'Selection...'

    within('.selection-action-container') do
      assert page.has_text?("Compare selected"), "Compare selected link text not found"
      assert page.has_link?("Copy selected"), "Copy selected link not found"
      assert page.has_link?("Move selected"), "Move selected link not found"
      assert page.has_link?("Remove selected"), "Remove selected link not found"

      click_link "#{action} selected"
    end

    # select the destination project if a Copy or Move action is being performed
    if action == 'Copy' || action == 'Move'
      within(".modal-container") do
        find('.selectable', text: dest['name']).click
        find('.modal-footer a,button', text: action).click
        wait_for_ajax
      end
    end
  end

  # Test copy action state. It should not be available when a subproject is selected.
  test "copy action is disabled when a subproject is selected" do
    my_project = api_fixture('groups')['aproject']
    my_collection = api_fixture('collections')['collection_to_move_around_in_aproject']
    my_subproject = api_fixture('groups')['asubproject']

    # verify that selection options are disabled on the project until an item is selected
    visit page_with_token 'active', '/'
    find('.arv-project-list a,button', text: my_project['name']).click

    click_button 'Selection...'
    within('.selection-action-container') do
      page.assert_selector 'li.disabled', text: 'Combine selected collections into a new collection'
      page.assert_selector 'li.disabled', text: 'Compare selected'
      page.assert_selector 'li.disabled', text: 'Copy selected'
      page.assert_selector 'li.disabled', text: 'Move selected'
      page.assert_selector 'li.disabled', text: 'Remove selected'
    end

    # select collection and verify links are enabled
    visit page_with_token 'active', '/'
    find('.arv-project-list a,button', text: my_project['name']).click
    assert page.has_text?(my_collection['name']), 'Collection not found in project'

    within('tr', text: my_collection['name']) do
      find('input[type=checkbox]').click
    end

    click_button 'Selection...'
    within('.selection-action-container') do
      page.assert_no_selector 'li.disabled', text: 'Combine selected collections into a new collection'
      page.assert_selector 'li', text: 'Combine selected collections into a new collection'
      page.assert_selector 'li.disabled', text: 'Compare selected'
      page.assert_no_selector 'li.disabled', text: 'Copy selected'
      page.assert_selector 'li', text: 'Copy selected'
      page.assert_no_selector 'li.disabled', text: 'Move selected'
      page.assert_selector 'li', text: 'Move selected'
      page.assert_no_selector 'li.disabled', text: 'Remove selected'
      page.assert_selector 'li', text: 'Remove selected'
    end

    # select subproject and verify that copy action is disabled
    visit page_with_token 'active', '/'
    find('.arv-project-list a,button', text: my_project['name']).click

    click_link 'Subprojects'
    assert page.has_text?(my_subproject['name']), 'Subproject not found in project'

    within('tr', text: my_subproject['name']) do
      find('input[type=checkbox]').click
    end

    click_button 'Selection...'
    within('.selection-action-container') do
      page.assert_selector 'li.disabled', text: 'Combine selected collections into a new collection'
      page.assert_selector 'li.disabled', text: 'Compare selected'
      page.assert_selector 'li.disabled', text: 'Copy selected'
      page.assert_no_selector 'li.disabled', text: 'Move selected'
      page.assert_selector 'li', text: 'Move selected'
      page.assert_no_selector 'li.disabled', text: 'Remove selected'
      page.assert_selector 'li', text: 'Remove selected'
    end

    # select subproject and a collection and verify that copy action is still disabled
    visit page_with_token 'active', '/'
    find('.arv-project-list a,button', text: my_project['name']).click

    click_link 'Subprojects'
    assert page.has_text?(my_subproject['name']), 'Subproject not found in project'

    within('tr', text: my_subproject['name']) do
      find('input[type=checkbox]').click
    end

    click_link 'Data collections'
    assert page.has_text?(my_collection['name']), 'Collection not found in project'

    within('tr', text: my_collection['name']) do
      find('input[type=checkbox]').click
    end

    click_button 'Selection...'
    within('.selection-action-container') do
      page.assert_selector 'li.disabled', text: 'Combine selected collections into a new collection'
      page.assert_selector 'li.disabled', text: 'Compare selected'
      page.assert_selector 'li.disabled', text: 'Copy selected'
      page.assert_no_selector 'li.disabled', text: 'Move selected'
      page.assert_selector 'li', text: 'Move selected'
      page.assert_no_selector 'li.disabled', text: 'Remove selected'
      page.assert_selector 'li', text: 'Remove selected'
    end
  end

  test "combine selected collections into new collection" do
    my_project = api_fixture('groups')['aproject']
    my_collection = api_fixture('collections')['collection_to_move_around_in_aproject']

    visit page_with_token 'active', '/'
    find('.arv-project-list a,button', text: my_project['name']).click
    assert page.has_text?(my_collection['name']), 'Collection not found in project'

    within('tr', text: my_collection['name']) do
      find('input[type=checkbox]').click
    end

    click_button 'Selection...'
    within('.selection-action-container') do
      click_link 'Combine selected collections into a new collection'
    end

    # back in project page
    assert page.has_text?(my_collection['name']), 'Collection not found in project'
    assert page.has_link?('Jobs and pipelines'), 'Jobs and pipelines link not found in project'
  end

end
