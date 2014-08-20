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
    click_link 'Other objects'
    within '.selection-action-container' do
      # Wait for the tab to load:
      assert_selector 'tr[data-kind="arvados#specimen"]'
      within first('tr', text: 'Specimen') do
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
end
