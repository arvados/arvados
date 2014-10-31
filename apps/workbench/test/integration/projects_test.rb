require 'integration_helper'
require 'selenium-webdriver'
require 'headless'

class ProjectsTest < ActionDispatch::IntegrationTest
  setup do
    Capybara.current_driver = Capybara.javascript_driver
  end

  test 'Check collection count for A Project in the tab pane titles' do
    project_uuid = api_fixture('groups')['aproject']['uuid']
    visit page_with_token 'active', '/projects/' + project_uuid
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
    assert(has_no_text?('.container-fluid', text: '*Textile description for A project*'),
           "Description is not rendered properly")
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
      page.assert_no_selector '.editable', text: 'Now I have a name.'
    end
  end

  test 'Create a project and move it into a different project' do
    visit page_with_token 'active', '/projects'
    find("#projects-menu").click
    find(".dropdown-menu a", text: "Home").click
    find('.btn', text: "Add a subproject").click

    # within('.editable', text: 'New project') do
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

  def show_project_using(auth_key, proj_key='aproject')
    project_uuid = api_fixture('groups')[proj_key]['uuid']
    visit(page_with_token(auth_key, "/projects/#{project_uuid}"))
    assert(page.has_text?("A Project"), "not on expected project page")
  end

  def share_rows
    find('#project_sharing').all('tr')
  end

  def add_share_and_check(share_type, name, obj=nil)
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
      if share_type == 'users' and obj and obj['email']
        assert(page.has_text?(obj['email']), "Did not find user's email")
      end
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
    add_share_and_check("users", new_name, add_user)
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
    ['Move',api_fixture('collections')['collection_to_move_around_in_aproject'],
      api_fixture('groups')['aproject'],api_fixture('groups')['asubproject']],
    ['Remove',api_fixture('collections')['collection_to_move_around_in_aproject'],
      api_fixture('groups')['aproject']],
    ['Copy',api_fixture('collections')['collection_to_move_around_in_aproject'],
      api_fixture('groups')['aproject'],api_fixture('groups')['asubproject']],
    ['Remove',api_fixture('collections')['collection_in_aproject_with_same_name_as_in_home_project'],
      api_fixture('groups')['aproject'],nil,true],
  ].each do |action, my_collection, src, dest=nil, expect_name_change=nil|
    test "selection #{action} #{expect_name_change} for project" do
      perform_selection_action src, dest, my_collection, action

      case action
      when 'Copy'
        assert page.has_text?(my_collection['name']), 'Collection not found in src project after copy'
        visit page_with_token 'active', '/'
        find("#projects-menu").click
        find(".dropdown-menu a", text: dest['name']).click
        assert page.has_text?(my_collection['name']), 'Collection not found in dest project after copy'

        # now remove it from destination project to restore to original state
        perform_selection_action dest, nil, my_collection, 'Remove'
      when 'Move'
        assert page.has_no_text?(my_collection['name']), 'Collection still found in src project after move'
        visit page_with_token 'active', '/'
        find("#projects-menu").click
        find(".dropdown-menu a", text: dest['name']).click
        assert page.has_text?(my_collection['name']), 'Collection not found in dest project after move'

        # move it back to src project to restore to original state
        perform_selection_action dest, src, my_collection, action
      when 'Remove'
        assert page.has_no_text?(my_collection['name']), 'Collection still found in src project after remove'
        visit page_with_token 'active', '/'
        find("#projects-menu").click
        find(".dropdown-menu a", text: "Home").click
        assert page.has_text?(my_collection['name']), 'Collection not found in home project after remove'
        if expect_name_change
          assert page.has_text?(my_collection['name']+' removed from ' + src['name']),
            'Collection with update name is not found in home project after remove'
        end
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
    find("#projects-menu").click
    find(".dropdown-menu a", text: my_project['name']).click

    click_button 'Selection...'
    within('.selection-action-container') do
      page.assert_selector 'li.disabled', text: 'Create new collection with selected collections'
      page.assert_selector 'li.disabled', text: 'Compare selected'
      page.assert_selector 'li.disabled', text: 'Copy selected'
      page.assert_selector 'li.disabled', text: 'Move selected'
      page.assert_selector 'li.disabled', text: 'Remove selected'
    end

    # select collection and verify links are enabled
    visit page_with_token 'active', '/'
    find("#projects-menu").click
    find(".dropdown-menu a", text: my_project['name']).click
    assert page.has_text?(my_collection['name']), 'Collection not found in project'

    within('tr', text: my_collection['name']) do
      find('input[type=checkbox]').click
    end

    click_button 'Selection...'
    within('.selection-action-container') do
      page.assert_no_selector 'li.disabled', text: 'Create new collection with selected collections'
      page.assert_selector 'li', text: 'Create new collection with selected collections'
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
    find("#projects-menu").click
    find(".dropdown-menu a", text: my_project['name']).click

    click_link 'Subprojects'
    assert page.has_text?(my_subproject['name']), 'Subproject not found in project'

    within('tr', text: my_subproject['name']) do
      find('input[type=checkbox]').click
    end

    click_button 'Selection...'
    within('.selection-action-container') do
      page.assert_selector 'li.disabled', text: 'Create new collection with selected collections'
      page.assert_selector 'li.disabled', text: 'Compare selected'
      page.assert_selector 'li.disabled', text: 'Copy selected'
      page.assert_no_selector 'li.disabled', text: 'Move selected'
      page.assert_selector 'li', text: 'Move selected'
      page.assert_no_selector 'li.disabled', text: 'Remove selected'
      page.assert_selector 'li', text: 'Remove selected'
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

    click_button 'Selection...'
    within('.selection-action-container') do
      page.assert_selector 'li.disabled', text: 'Create new collection with selected collections'
      page.assert_selector 'li.disabled', text: 'Compare selected'
      page.assert_selector 'li.disabled', text: 'Copy selected'
      page.assert_no_selector 'li.disabled', text: 'Move selected'
      page.assert_selector 'li', text: 'Move selected'
      page.assert_no_selector 'li.disabled', text: 'Remove selected'
      page.assert_selector 'li', text: 'Remove selected'
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

      click_button 'Selection...'
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

  [
    ['project with 10 collections', 10],
    ['project with 201 collections', 201], # two pages of data
  ].each do |project_name, amount|
    test "scroll collections tab for #{project_name} with #{amount} objects" do
      headless = Headless.new
      headless.start
      Capybara.current_driver = :selenium

      visit page_with_token 'user1_with_load'

      find("#projects-menu").click
      find(".dropdown-menu a", text: project_name).click

      my_collections = []
      for i in 1..amount
        my_collections << "Collection_#{i}"
      end

      # verify Data collections scroll
      assert(page.has_text?("Data collections (#{amount})"), "Number of collections did not match the input amount")

      click_link 'Data collections'
      begin
        wait_for_ajax
      rescue
      end

      verify_collections = my_collections.dup
      unexpected_items = []
      collections_count = 0
      within('.arv-project-Data_collections') do
        page.execute_script "window.scrollBy(0,999000)"
        begin
          wait_for_ajax
        rescue
        end

        # Visit all rows. If not all expected collections are found, retry
        found_collections = page.all('tr[data-kind="arvados#collection"]')
        collections_count = found_collections.count

        (0..collections_count-1).each do |i|
          # Found row text would be of the format "Show Collection_#{n} "
          collection_name = found_collections[i].text.split[1]
          if !my_collections.include? collection_name
            unexpected_items << collection_name
          else
            verify_collections.delete collection_name
          end
        end

        assert_equal true, unexpected_items.empty?, "Found unexpected items #{unexpected_items.inspect}"
        assert_equal amount, collections_count, "Found different number of collections"
        assert_equal true, verify_collections.empty?, "Did not find all the collections"
      end
    end
  end

  [
    ['project with 10 collections', 10],
    ['project with 201 collections', 201], # two pages of data
  ].each do |project_name, amount|
    test "scroll collections tab for #{project_name} with #{amount} objects, with ascending sort (case insensitive)" do
      headless = Headless.new
      headless.start
      Capybara.current_driver = :selenium

      visit page_with_token 'user1_with_load'

      find("#projects-menu").click
      find(".dropdown-menu a", text: project_name).click

      my_collections = []
      for i in 1..amount
        my_collections << "Collection_#{i}"
      end

      # verify Data collections scroll
      assert(page.has_text?("Data collections (#{amount})"), "Number of collections did not match the input amount")

      click_link 'Data collections'
      begin
        wait_for_ajax
      rescue
      end

      find('th[data-sort-order="collections.name"]').click
      wait_for_ajax

      verify_collections = my_collections.dup
      unexpected_items = []
      collections_count = 0
      within('.arv-project-Data_collections') do
        page.execute_script "window.scrollBy(0,999000)"
        begin
          wait_for_ajax
        rescue
        end

        # Visit all rows. If not all expected collections are found, retry
        found_collections = page.all('tr[data-kind="arvados#collection"]')
        collections_count = found_collections.count

        previous = nil
        (0..collections_count-1).each do |i|
          # Found row text would be of the format "Show Collection_#{n} "
          collection_name = found_collections[i].text.split[1]
          if !my_collections.include? collection_name
            unexpected_items << collection_name
          else
            verify_collections.delete collection_name
          end
          # check sort order
          assert_operator( previous.downcase, :<=, collection_name.downcase) if previous
          previous = collection_name
        end

        assert_equal true, unexpected_items.empty?, "Found unexpected items #{unexpected_items.inspect}"
        assert_equal amount, collections_count, "Found different number of collections"
        assert_equal true, verify_collections.empty?, "Did not find all the collections"
      end
    end
  end

  [
    ['project with 10 pipelines', 10, 0],
    ['project with 2 pipelines and 60 jobs', 2, 60],
    ['project with 25 pipelines', 25, 0],
  ].each do |project_name, num_pipelines, num_jobs|
    test "scroll pipeline instances tab for #{project_name} with #{num_pipelines} pipelines and #{num_jobs} jobs" do
      headless = Headless.new
      headless.start
      Capybara.current_driver = :selenium

      visit page_with_token 'user1_with_load'

      find("#projects-menu").click
      find(".dropdown-menu a", text: project_name).click

      my_pipelines = []
      (0..num_pipelines-1).each do |i|
        name = "pipeline_#{i}"
        my_pipelines << name
      end

      # verify Jobs and pipelines tab scroll
      assert(page.has_text?("Jobs and pipelines (#{num_pipelines+num_jobs})"), "Number of objects did not match the input counts")
      click_link 'Jobs and pipelines'
      begin
        wait_for_ajax
      rescue
      end

      verify_pipelines = my_pipelines.dup
      unexpected_items = []
      object_count = 0
      within('.arv-project-Jobs_and_pipelines') do
        page.execute_script "window.scrollBy(0,999000)"
        begin
          wait_for_ajax
        rescue
        end

        # Visit all rows. Repeat if not all expected my_pipelines are found (inifinite scrolling should kick in)
        pipelines_found = page.all('tr[data-kind="arvados#pipelineInstance"]')
        found_pipeline_count = pipelines_found.count
        (0..found_pipeline_count-1).each do |i|
          name = pipelines_found[i].text.split[1]
          if !my_pipelines.include? name
            unexpected_items << name
          else
            verify_pipelines.delete name
          end

          assert_equal true, unexpected_items.empty?, "Found unexpected items #{unexpected_items.inspect}"
        end

        jobs_found = page.all('tr[data-kind="arvados#job"]')
        found_job_count = jobs_found.count

        assert_equal num_pipelines, found_pipeline_count, "Found different number of pipelines and jobs"
        assert_equal num_jobs, found_job_count, 'Did not find expected number of jobs'
        assert_equal true, verify_pipelines.empty?, "Did not find all the pipelines and jobs"
      end
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

end
