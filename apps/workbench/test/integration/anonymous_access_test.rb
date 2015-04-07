require 'integration_helper'

class AnonymousAccessTest < ActionDispatch::IntegrationTest
  # These tests don't do state-changing API calls. Save some time by
  # skipping the database reset.
  reset_api_fixtures :after_each_test, false
  reset_api_fixtures :after_suite, true

  setup do
    need_javascript
    Rails.configuration.anonymous_user_token = api_fixture('api_client_authorizations')['anonymous']['api_token']
  end

  PUBLIC_PROJECT = "/projects/#{api_fixture('groups')['anonymously_accessible_project']['uuid']}"

  def verify_site_navigation_anonymous_enabled user, is_active
    if user
      if user['is_active']
        assert_text 'Unrestricted public data'
        assert_selector 'a', text: 'Projects'
      else
        assert_text 'indicate that you have read and accepted the user agreement'
      end
      within('.navbar-fixed-top') do
        assert_selector 'a', text: Rails.configuration.site_name.downcase
        assert_selector 'a', text: "#{user['email']}"
        find('a', text: "#{user['email']}").click
        within('.dropdown-menu') do
          assert_selector 'a', text: 'Log out'
        end
      end
    else  # anonymous
      assert_text 'Unrestricted public data'
      within('.navbar-fixed-top') do
        assert_text Rails.configuration.site_name.downcase
        assert_no_selector 'a', text: Rails.configuration.site_name.downcase
        assert_selector 'a', text: 'Log in'
      end
    end
  end

  [
    [nil, nil, false, false],
    ['inactive', api_fixture('users')['inactive'], false, false],
    ['active', api_fixture('users')['active'], true, true],
  ].each do |token, user, is_active|
    test "visit public project as user #{token.inspect} when anonymous browsing is enabled" do
      if !token
        visit PUBLIC_PROJECT
      else
        visit page_with_token(token, PUBLIC_PROJECT)
      end

      verify_site_navigation_anonymous_enabled user, is_active
    end
  end

  test "selection actions when anonymous user accesses shared project" do
    visit PUBLIC_PROJECT

    assert_selector 'a', text: 'Description'
    assert_selector 'a', text: 'Data collections'
    assert_selector 'a', text: 'Jobs and pipelines'
    assert_selector 'a', text: 'Pipeline templates'
    assert_selector 'a', text: 'Advanced'
    assert_no_selector 'a', text: 'Subprojects'
    assert_no_selector 'a', text: 'Other objects'
    assert_no_selector 'button', text: 'Add data'

    click_link 'Data collections'
    click_button 'Selection'
    within('.selection-action-container') do
      assert_selector 'li', text: 'Compare selected'
      assert_no_selector 'li', text: 'Create new collection with selected collections'
      assert_no_selector 'li', text: 'Copy selected'
      assert_no_selector 'li', text: 'Move selected'
      assert_no_selector 'li', text: 'Remove selected'
    end
  end

  test "anonymous user accesses data collections tab in shared project" do
    visit PUBLIC_PROJECT
    click_link 'Data collections'
    collection = api_fixture('collections')['user_agreement_in_anonymously_accessible_project']
    assert_text 'GNU General Public License'

    assert_selector 'a', text: 'Data collections'

    # click on show collection
    within "tr[data-object-uuid=\"#{collection['uuid']}\"]" do
      click_link 'Show'
    end

    # in collection page
    assert_no_selector 'input', text: 'Create sharing link'
    assert_no_text 'Sharing and permissions'
    assert_no_selector 'a', text: 'Upload'
    assert_no_selector 'button', 'Selection'

    within '#collection_files tr,li', text: 'GNU_General_Public_License,_version_3.pdf' do
      assert page.has_no_selector?('[value*="GNU_General_Public_License"]')
      find 'a[title~=View]'
      find 'a[title~=Download]'
    end
  end

  test 'view file' do
    magic = rand(2**512).to_s 36
    CollectionsController.any_instance.stubs(:file_enumerator).returns([magic])
    collection = api_fixture('collections')['public_text_file']
    visit '/collections/' + collection['uuid']
    find('tr,li', text: 'Hello world.txt').
      find('a[title~=View]').click
    assert_text magic
  end

  [
    'running_job',
    'completed_job',
    'pipelineInstance'
  ].each do |type|
    test "anonymous user accesses jobs and pipelines tab in shared project and clicks on #{type}" do
      visit PUBLIC_PROJECT
      click_link 'Data collections'
      assert_text 'GNU General Public License'

      click_link 'Jobs and pipelines'
      assert_text 'Pipeline in publicly accessible project'

      # click on the specified job
      if type.include? 'job'
        verify_job_row type
      else
        verify_pipeline_instance_row
      end
    end
  end

  def verify_job_row look_for
    within first('tr', text: look_for) do
      click_link 'Show'
    end
    assert_text 'Public Projects Unrestricted public data'
    assert_text 'script_version'

    assert_text 'zzzzz-tpzed-xurymjxw79nv3jz' # modified by user
    assert_no_selector 'a', text: 'zzzzz-tpzed-xurymjxw79nv3jz'
    assert_no_selector 'a', text: 'Move job'
    assert_no_selector 'button', text: 'Cancel'
    assert_no_selector 'button', text: 'Re-run job'
  end

  def verify_pipeline_instance_row
    within first('tr[data-kind="arvados#pipelineInstance"]') do
      assert_text 'Pipeline in publicly accessible project'
      click_link 'Show'
    end

    # in pipeline instance page
    assert_text 'Public Projects Unrestricted public data'
    assert_text 'This pipeline is complete'
    assert_no_selector 'a', text: 'Re-run with latest'
    assert_no_selector 'a', text: 'Re-run options'
  end

  test "anonymous user accesses pipeline templates tab in shared project" do
    visit PUBLIC_PROJECT
    click_link 'Data collections'
    assert_text 'GNU General Public License'

    assert_selector 'a', text: 'Pipeline templates'

    click_link 'Pipeline templates'
    assert_text 'Pipeline template in publicly accessible project'

    within first('tr[data-kind="arvados#pipelineTemplate"]') do
      click_link 'Show'
    end

    # in template page
    assert_text 'Public Projects Unrestricted public data'
    assert_text 'script version'
    assert_no_selector 'a', text: 'Run this pipeline'
  end

  [
    ['pipeline_in_publicly_accessible_project', true],
    ['pipeline_in_publicly_accessible_project_but_other_objects_elsewhere', false],
    ['pipeline_in_publicly_accessible_project_but_other_objects_elsewhere', false, 'admin'],

    ['completed_job_in_publicly_accessible_project', true],
    ['job_in_publicly_accessible_project_but_other_objects_elsewhere', false],
  ].each do |fixture, objects_readable, user=nil|
    test "access #{fixture} in public project with objects readable=#{objects_readable} with user #{user}" do
      pipeline_page = true if fixture.include?('pipeline')

      if pipeline_page
        object = api_fixture('pipeline_instances')[fixture]
        page = "/pipeline_instances/#{object['uuid']}"
      else      # job
        object = api_fixture('jobs')[fixture]
        page = "/jobs/#{object['uuid']}"
      end

      if user
        visit page_with_token user, page
      else
        visit page
      end

      # click job link, if in pipeline page
      click_link 'foo' if pipeline_page

      if objects_readable or user
        if pipeline_page
          assert_text 'This pipeline was created from'
          assert_selector 'a', object['components']['foo']['job']['uuid']
        end
        assert_no_text 'Output data not available'
        assert_selector 'a[href="#Log"]', text: 'Log'
        assert_no_selector 'a[data-toggle="disabled"]', text: 'Log'
      else
        if pipeline_page
          assert_no_text 'This pipeline was created from'  # template is not readable
          assert_no_selector 'a', text: object['components']['foo']['job']['uuid']
        end
        assert_text 'Output data not available'
        assert_text object['job']
        assert_selector 'a[href="#Log"]', text: 'Log'
        assert_selector 'a[data-toggle="disabled"]', text: 'Log'
      end

      click_link 'Log'
      if objects_readable or user
        assert_no_text 'foo'  # should be in Log tab
        assert_text 'stderr crunchstat'   if pipeline_page
      else
        assert_text 'foo'     # Log tab disabled and hence still in first tab
        assert_no_text 'stderr crunchstat'  # log line shouldn't be seen
      end
    end
  end

  test "anonymous user accesses collection in shared project" do
    visit "/collections/#{api_fixture('collections')['public_text_file']['uuid']}"

    # in collection page
    assert_text 'Public Projects Unrestricted public data'
    assert_text 'Hello world'
    assert_text 'Content address'
    assert_selector 'a', text: 'Provenance graph'
  end
end
