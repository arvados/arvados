require 'integration_helper'
require 'helpers/share_object_helper'

class ProjectsTest < ActionDispatch::IntegrationTest
  include ShareObjectHelper

  setup do
    need_javascript
  end

  test 'Check collection count for A Project in the tab pane titles' do
    project_uuid = api_fixture('groups')['aproject']['uuid']
    visit page_with_token 'active', '/projects/' + project_uuid
    wait_for_ajax
    collection_count = page.all("[data-pk*='collection']").count
    assert_selector '#Data_collections-tab span', text: "(#{collection_count})"
  end

  test 'Find a project and edit its description' do
    visit page_with_token 'active', '/'
    find("#projects-menu").click
    find(".dropdown-menu a", text: "A Project").click
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
    find("#projects-menu").click
    find(".dropdown-menu a", text: "A Project").click
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
    assert_no_text '*Textile description for A project*'
    assert(find?('.container-fluid', text: 'Textile description for A project'),
           "Description update did not survive page refresh")
    assert(find?('.container-fluid', text: 'And a new paragraph in description'),
           "Description did not contain the expected new paragraph")
    assert(page.has_link?("take me home"), "link not found in description")

    click_link 'take me home'

    # now in dashboard
    assert(page.has_text?('Active pipelines'), 'Active pipelines - not found on dashboard')
  end

  test 'Find a project and edit description to html description' do
    visit page_with_token 'active', '/'
    find("#projects-menu").click
    find(".dropdown-menu a", text: "A Project").click
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
    assert page.has_text?('Active pipelines')
  end

  test 'Find a project and edit description to textile description with link to object' do
    visit page_with_token 'active', '/'
    find("#projects-menu").click
    find(".dropdown-menu a", text: "A Project").click
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
      assert_no_selector '.editable', text: 'Now I have a name.'
    end
  end

  test 'Create a project and move it into a different project' do
    visit page_with_token 'active', '/projects'
    find("#projects-menu").click
    find(".dropdown-menu a", text: "Home").click
    find('.btn', text: "Add a subproject").click

    within('h2') do
      find('.fa-pencil').click
      find('.editable-input input').set('Project 1234')
      find('.glyphicon-ok').click
    end
    wait_for_ajax

    visit '/projects'
    find("#projects-menu").click
    find(".dropdown-menu a", text: "Home").click
    find('.btn', text: "Add a subproject").click
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

  def open_groups_sharing(project_name="aproject", token_name="active")
    project = api_fixture("groups", project_name)
    visit(page_with_token(token_name, "/projects/#{project['uuid']}"))
    click_on "Sharing"
    click_on "Share with groups"
  end

  def group_name(group_key)
    api_fixture("groups", group_key, "name")
  end

  test "projects not publicly sharable when anonymous browsing disabled" do
    Rails.configuration.anonymous_user_token = false
    open_groups_sharing
    # Check for a group we do expect first, to make sure the modal's loaded.
    assert_selector(".modal-container .selectable",
                    text: group_name("all_users"))
    assert_no_selector(".modal-container .selectable",
                       text: group_name("anonymous_group"))
  end

  test "projects publicly sharable when anonymous browsing enabled" do
    Rails.configuration.anonymous_user_token = "testonlytoken"
    open_groups_sharing
    assert_selector(".modal-container .selectable",
                    text: group_name("anonymous_group"))
  end

  test "project viewer can't see project sharing tab" do
    show_object_using('project_viewer', 'groups', 'aproject', 'A Project')
    assert(page.has_no_link?("Sharing"),
           "read-only project user sees sharing tab")
  end

  test "project owner can manage sharing for another user" do
    add_user = api_fixture('users')['future_project_user']
    new_name = ["first_name", "last_name"].map { |k| add_user[k] }.join(" ")

    show_object_using('active', 'groups', 'aproject', 'A Project')
    click_on "Sharing"
    add_share_and_check("users", new_name, add_user)
    modify_share_and_check(new_name)
  end

  test "project owner can manage sharing for another group" do
    new_name = api_fixture('groups')['future_project_viewing_group']['name']

    show_object_using('active', 'groups', 'aproject', 'A Project')
    click_on "Sharing"
    add_share_and_check("groups", new_name)
    modify_share_and_check(new_name)
  end

  test "'share with group' listing does not offer projects" do
    show_object_using('active', 'groups', 'aproject', 'A Project')
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
    ['Move',api_fixture('collections')['collection_to_move_around_in_aproject'],
      api_fixture('groups')['aproject'],api_fixture('groups')['asubproject']],
    ['Remove',api_fixture('collections')['collection_to_move_around_in_aproject'],
      api_fixture('groups')['aproject']],
    ['Copy',api_fixture('collections')['collection_to_move_around_in_aproject'],
      api_fixture('groups')['aproject'],api_fixture('groups')['asubproject']],
    ['Remove',api_fixture('collections')['collection_in_aproject_with_same_name_as_in_home_project'],
      api_fixture('groups')['aproject'],nil,true],
  ].each do |action, my_collection, src, dest=nil, expect_name_change=nil|
    test "selection #{action} -> #{expect_name_change.inspect} for project" do
      perform_selection_action src, dest, my_collection, action

      case action
      when 'Copy'
        assert page.has_text?(my_collection['name']), 'Collection not found in src project after copy'
        visit page_with_token 'active', '/'
        find("#projects-menu").click
        find(".dropdown-menu a", text: dest['name']).click
        assert page.has_text?(my_collection['name']), 'Collection not found in dest project after copy'

      when 'Move'
        assert page.has_no_text?(my_collection['name']), 'Collection still found in src project after move'
        visit page_with_token 'active', '/'
        find("#projects-menu").click
        find(".dropdown-menu a", text: dest['name']).click
        assert page.has_text?(my_collection['name']), 'Collection not found in dest project after move'

      when 'Remove'
        assert page.has_no_text?(my_collection['name']), 'Collection still found in src project after remove'
      end
    end
  end

  def perform_selection_action src, dest, item, action
    visit page_with_token 'active', '/'
    find("#projects-menu").click
    find(".dropdown-menu a", text: src['name']).click
    assert page.has_text?(item['name']), 'Collection not found in src project'

    within('tr', text: item['name']) do
      find('input[type=checkbox]').click
    end

    click_button 'Selection'

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
    find("#projects-menu").click
    find(".dropdown-menu a", text: my_project['name']).click

    click_button 'Selection'
    within('.selection-action-container') do
      assert_selector 'li.disabled', text: 'Create new collection with selected collections'
      assert_selector 'li.disabled', text: 'Compare selected'
      assert_selector 'li.disabled', text: 'Copy selected'
      assert_selector 'li.disabled', text: 'Move selected'
      assert_selector 'li.disabled', text: 'Remove selected'
    end

    # select collection and verify links are enabled
    visit page_with_token 'active', '/'
    find("#projects-menu").click
    find(".dropdown-menu a", text: my_project['name']).click
    assert page.has_text?(my_collection['name']), 'Collection not found in project'

    within('tr', text: my_collection['name']) do
      find('input[type=checkbox]').click
    end

    click_button 'Selection'
    within('.selection-action-container') do
      assert_no_selector 'li.disabled', text: 'Create new collection with selected collections'
      assert_selector 'li', text: 'Create new collection with selected collections'
      assert_selector 'li.disabled', text: 'Compare selected'
      assert_no_selector 'li.disabled', text: 'Copy selected'
      assert_selector 'li', text: 'Copy selected'
      assert_no_selector 'li.disabled', text: 'Move selected'
      assert_selector 'li', text: 'Move selected'
      assert_no_selector 'li.disabled', text: 'Remove selected'
      assert_selector 'li', text: 'Remove selected'
    end

    # select subproject and verify that copy action is disabled
    visit page_with_token 'active', '/'
    find("#projects-menu").click
    find(".dropdown-menu a", text: my_project['name']).click

    click_link 'Subprojects'
    assert page.has_text?(my_subproject['name']), 'Subproject not found in project'

    within('tr', text: my_subproject['name']) do
      find('input[type=checkbox]').click
    end

    click_button 'Selection'
    within('.selection-action-container') do
      assert_selector 'li.disabled', text: 'Create new collection with selected collections'
      assert_selector 'li.disabled', text: 'Compare selected'
      assert_selector 'li.disabled', text: 'Copy selected'
      assert_no_selector 'li.disabled', text: 'Move selected'
      assert_selector 'li', text: 'Move selected'
      assert_no_selector 'li.disabled', text: 'Remove selected'
      assert_selector 'li', text: 'Remove selected'
    end

    # select subproject and a collection and verify that copy action is still disabled
    visit page_with_token 'active', '/'
    find("#projects-menu").click
    find(".dropdown-menu a", text: my_project['name']).click

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

    click_link 'Subprojects'
    click_button 'Selection'
    within('.selection-action-container') do
      assert_selector 'li.disabled', text: 'Create new collection with selected collections'
      assert_selector 'li.disabled', text: 'Compare selected'
      assert_selector 'li.disabled', text: 'Copy selected'
      assert_no_selector 'li.disabled', text: 'Move selected'
      assert_selector 'li', text: 'Move selected'
      assert_no_selector 'li.disabled', text: 'Remove selected'
      assert_selector 'li', text: 'Remove selected'
    end
  end

  # When project tabs are switched, only options applicable to the current tab's selections are enabled.
  test "verify selection options when tabs are switched" do
    my_project = api_fixture('groups')['aproject']
    my_collection = api_fixture('collections')['collection_to_move_around_in_aproject']
    my_subproject = api_fixture('groups')['asubproject']

    # select subproject and a collection and verify that copy action is still disabled
    visit page_with_token 'active', '/'
    find("#projects-menu").click
    find(".dropdown-menu a", text: my_project['name']).click

    # Select a sub-project
    click_link 'Subprojects'
    assert page.has_text?(my_subproject['name']), 'Subproject not found in project'

    within('tr', text: my_subproject['name']) do
      find('input[type=checkbox]').click
    end

    # Select a collection
    click_link 'Data collections'
    assert page.has_text?(my_collection['name']), 'Collection not found in project'

    within('tr', text: my_collection['name']) do
      find('input[type=checkbox]').click
    end

    # Go back to Subprojects tab
    click_link 'Subprojects'
    click_button 'Selection'
    within('.selection-action-container') do
      assert_selector 'li.disabled', text: 'Create new collection with selected collections'
      assert_selector 'li.disabled', text: 'Compare selected'
      assert_selector 'li.disabled', text: 'Copy selected'
      assert_no_selector 'li.disabled', text: 'Move selected'
      assert_selector 'li', text: 'Move selected'
      assert_no_selector 'li.disabled', text: 'Remove selected'
      assert_selector 'li', text: 'Remove selected'
    end

    # Close the dropdown by clicking outside it.
    find('.dropdown-toggle', text: 'Selection').find(:xpath, '..').click

    # Go back to Data collections tab
    find('.nav-tabs a', text: 'Data collections').click
    click_button 'Selection'
    within('.selection-action-container') do
      assert_no_selector 'li.disabled', text: 'Create new collection with selected collections'
      assert_selector 'li', text: 'Create new collection with selected collections'
      assert_selector 'li.disabled', text: 'Compare selected'
      assert_no_selector 'li.disabled', text: 'Copy selected'
      assert_selector 'li', text: 'Copy selected'
      assert_no_selector 'li.disabled', text: 'Move selected'
      assert_selector 'li', text: 'Move selected'
      assert_no_selector 'li.disabled', text: 'Remove selected'
      assert_selector 'li', text: 'Remove selected'
    end
  end

  # "Move selected" and "Remove selected" options should not be available when current user cannot write to the project
  test "move selected and remove selected actions not available when current user cannot write to project" do
    my_project = api_fixture('groups')['anonymously_accessible_project']
    visit page_with_token 'active', "/projects/#{my_project['uuid']}"

    click_button 'Selection'
    within('.selection-action-container') do
      assert_selector 'li', text: 'Create new collection with selected collections'
      assert_selector 'li', text: 'Compare selected'
      assert_selector 'li', text: 'Copy selected'
      assert_no_selector 'li', text: 'Move selected'
      assert_no_selector 'li', text: 'Remove selected'
    end
  end

  [
    ['active', true],
    ['project_viewer', false],
  ].each do |user, expect_collection_in_aproject|
    test "combine selected collections into new collection #{user} #{expect_collection_in_aproject}" do
      my_project = api_fixture('groups')['aproject']
      my_collection = api_fixture('collections')['collection_to_move_around_in_aproject']

      visit page_with_token user, '/'
      find("#projects-menu").click
      find(".dropdown-menu a", text: my_project['name']).click
      assert page.has_text?(my_collection['name']), 'Collection not found in project'

      within('tr', text: my_collection['name']) do
        find('input[type=checkbox]').click
      end

      click_button 'Selection'
      within('.selection-action-container') do
        click_link 'Create new collection with selected collections'
      end

      # now in the new collection page
      if expect_collection_in_aproject
        assert page.has_text?("Created new collection in the project #{my_project['name']}"),
                              'Not found flash message that new collection is created in aproject'
      else
        assert page.has_text?("Created new collection in your Home project"),
                              'Not found flash message that new collection is created in Home project'
      end
      assert page.has_text?('Content hash'), 'Not found content hash in collection page'
    end
  end

  [
    ["jobs", "/jobs"],
    ["pipelines", "/pipeline_instances"],
    ["collections", "/collections"]
  ].each do |target,path|
    test "Test dashboard button all #{target}" do
      visit page_with_token 'active', '/'
      click_link "All #{target}"
      assert_equal path, current_path
    end
  end

  def scroll_setup(project_name,
                   total_nbr_items,
                   item_list_parameter,
                   sorted = false,
                   sort_parameters = nil)
    project_uuid = api_fixture('groups')[project_name]['uuid']
    visit page_with_token 'user1_with_load', '/projects/' + project_uuid

    assert(page.has_text?("#{item_list_parameter.humanize} (#{total_nbr_items})"), "Number of #{item_list_parameter.humanize} did not match the input amount")

    click_link item_list_parameter.humanize
    wait_for_ajax

    if sorted
      find("th[data-sort-order='#{sort_parameters.gsub(/\s/,'')}']").click
      wait_for_ajax
    end
  end

  def scroll_items_check(nbr_items,
                         fixture_prefix,
                         item_list_parameter,
                         item_selector,
                         sorted = false)
    items = []
    for i in 1..nbr_items
      items << "#{fixture_prefix}#{i}"
    end

    verify_items = items.dup
    unexpected_items = []
    item_count = 0
    within(".arv-project-#{item_list_parameter}") do
      page.execute_script "window.scrollBy(0,999000)"
      begin
        wait_for_ajax
      rescue
      end

      # Visit all rows. If not all expected items are found, retry
      found_items = page.all(item_selector)
      item_count = found_items.count

      previous = nil
      (0..item_count-1).each do |i|
        # Found row text using the fixture string e.g. "Show Collection_#{n} "
        item_name = found_items[i].text.split[1]
        if !items.include? item_name
          unexpected_items << item_name
        else
          verify_items.delete item_name
        end
        if sorted
          # check sort order
          assert_operator( previous.downcase, :<=, item_name.downcase) if previous
          previous = item_name
        end
      end

      assert_equal true, unexpected_items.empty?, "Found unexpected #{item_list_parameter.humanize} #{unexpected_items.inspect}"
      assert_equal nbr_items, item_count, "Found different number of #{item_list_parameter.humanize}"
      assert_equal true, verify_items.empty?, "Did not find all the #{item_list_parameter.humanize}"
    end
  end

  [
    ['project_with_10_collections', 10],
    ['project_with_201_collections', 201], # two pages of data
  ].each do |project_name, nbr_items|
    test "scroll collections tab for #{project_name} with #{nbr_items} objects" do
      item_list_parameter = "Data_collections"
      scroll_setup project_name,
                   nbr_items,
                   item_list_parameter
      scroll_items_check nbr_items,
                         "Collection_",
                         item_list_parameter,
                         'tr[data-kind="arvados#collection"]'
    end
  end

  [
    ['project_with_10_collections', 10],
    ['project_with_201_collections', 201], # two pages of data
  ].each do |project_name, nbr_items|
    test "scroll collections tab for #{project_name} with #{nbr_items} objects with ascending sort (case insensitive)" do
      item_list_parameter = "Data_collections"
      scroll_setup project_name,
                   nbr_items,
                   item_list_parameter,
                   true,
                   "collections.name"
      scroll_items_check nbr_items,
                         "Collection_",
                         item_list_parameter,
                         'tr[data-kind="arvados#collection"]',
                         true
    end
  end

  [
    ['project_with_10_pipelines', 10, 0],
    ['project_with_2_pipelines_and_60_jobs', 2, 60],
    ['project_with_25_pipelines', 25, 0],
  ].each do |project_name, num_pipelines, num_jobs|
    test "scroll pipeline instances tab for #{project_name} with #{num_pipelines} pipelines and #{num_jobs} jobs" do
      item_list_parameter = "Jobs_and_pipelines"
      scroll_setup project_name,
                   num_pipelines + num_jobs,
                   item_list_parameter
      # check the general scrolling and the pipelines
      scroll_items_check num_pipelines,
                         "pipeline_",
                         item_list_parameter,
                         'tr[data-kind="arvados#pipelineInstance"]'
      # Check job count separately
      jobs_found = page.all('tr[data-kind="arvados#job"]')
      found_job_count = jobs_found.count
      assert_equal num_jobs, found_job_count, 'Did not find expected number of jobs'
    end
  end

  # Move button accessibility
  [
    ['admin', true],
    ['active', true],  # project owner
    ['project_viewer', false],
    ].each do |user, can_move|
    test "#{user} can move subproject under another user's Home #{can_move}" do
      project = api_fixture('groups')['aproject']
      collection = api_fixture('collections')['collection_to_move_around_in_aproject']

      # verify the project move button
      visit page_with_token user, "/projects/#{project['uuid']}"
      if can_move
        assert page.has_link? 'Move project...'
      else
        assert page.has_no_link? 'Move project...'
      end
    end
  end

  test "error while loading tab" do
    original_arvados_v1_base = Rails.configuration.arvados_v1_base

    visit page_with_token 'active', '/projects/' + api_fixture('groups')['aproject']['uuid']

    # Point to a bad api server url to generate error
    Rails.configuration.arvados_v1_base = "https://[100::f]:1/"
    click_link 'Other objects'
    within '#Other_objects' do
      # Error
      assert_selector('a', text: 'Reload tab')

      # Now point back to the orig api server and reload tab
      Rails.configuration.arvados_v1_base = original_arvados_v1_base
      click_link 'Reload tab'
      assert_no_selector('a', text: 'Reload tab')
      assert_selector('button', text: 'Selection')
      within '.selection-action-container' do
        assert_selector 'tr[data-kind="arvados#trait"]'
      end
    end
  end

  test "add new project using projects dropdown" do
    # verify that selection options are disabled on the project until an item is selected
    visit page_with_token 'active', '/'

    # Add a new project
    find("#projects-menu").click
    click_link 'Add a new project'
    assert_text 'New project'
    assert_text 'No description provided'

    # Add one more new project
    find("#projects-menu").click
    click_link 'Add a new project'
    match = /New project \(\d\)/.match page.text
    assert match, 'Expected project name not found'
    assert_text 'No description provided'
  end

  test "first tab loads data when visiting other tab directly" do
    # As of 2014-12-19, the first tab of project#show uses infinite scrolling.
    # Make sure that it loads data even if we visit another tab directly.
    need_selenium 'to land on specified tab using {url}#Advanced'
    project = api_fixture("groups", "aproject")
    visit(page_with_token("active_trustedclient",
                          "/projects/#{project['uuid']}#Advanced"))
    assert_text("API response")
    find("#page-wrapper .nav-tabs :first-child a").click
    assert_text("Collection modified at")
  end

  test "verify description column in data collections tab" do
    project = api_fixture('groups')['aproject']
    visit(page_with_token('active_trustedclient', "/projects/#{project['uuid']}"))

    collection = api_fixture('collections')['collection_to_move_around_in_aproject']
    assert_text collection['name']
    assert_text collection['description']
    assert_text 'Collection modified at' # there are collections with no descriptions
  end
end
