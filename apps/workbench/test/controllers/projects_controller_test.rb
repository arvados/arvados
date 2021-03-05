# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

require 'test_helper'
require 'helpers/share_object_helper'

class ProjectsControllerTest < ActionController::TestCase
  include ShareObjectHelper

  test "invited user is asked to sign user agreements on front page" do
    get :index, params: {}, session: session_for(:inactive)
    assert_response :redirect
    assert_match(/^#{Regexp.escape(user_agreements_url)}\b/,
                 @response.redirect_url,
                 "Inactive user was not redirected to user_agreements page")
  end

  test "uninvited user is asked to wait for activation" do
    get :index, params: {}, session: session_for(:inactive_uninvited)
    assert_response :redirect
    assert_match(/^#{Regexp.escape(inactive_users_url)}\b/,
                 @response.redirect_url,
                 "Uninvited user was not redirected to inactive user page")
  end

  [[:active, true],
   [:project_viewer, false]].each do |which_user, should_show|
    test "create subproject button #{'not ' unless should_show} shown to #{which_user}" do
      readonly_project_uuid = api_fixture('groups')['aproject']['uuid']
      get :show, params: {
        id: readonly_project_uuid
      }, session: session_for(which_user)
      buttons = css_select('[data-method=post]').select do |el|
        el.attributes['data-remote-href'].value.match /project.*owner_uuid.*#{readonly_project_uuid}/
      end
      if should_show
        assert_not_empty(buttons, "did not offer to create a subproject")
      else
        assert_empty(buttons.collect(&:to_s),
                     "offered to create a subproject in a non-writable project")
      end
    end
  end

  test "sharing a project with a user and group" do
    uuid_list = [api_fixture("groups")["future_project_viewing_group"]["uuid"],
                 api_fixture("users")["future_project_user"]["uuid"]]
    post(:share_with, params: {
           id: api_fixture("groups")["asubproject"]["uuid"],
           uuids: uuid_list,
           format: "json"},
         session: session_for(:active))
    assert_response :success
    assert_equal(uuid_list, json_response["success"])
  end

  test "user with project read permission can't add permissions" do
    share_uuid = api_fixture("users")["spectator"]["uuid"]
    post(:share_with, params: {
           id: api_fixture("groups")["aproject"]["uuid"],
           uuids: [share_uuid],
           format: "json"},
         session: session_for(:project_viewer))
    assert_response 422
    assert(json_response["errors"].andand.
             any? { |msg| msg.start_with?("#{share_uuid}: ") },
           "JSON response missing properly formatted sharing error")
  end

  test "admin can_manage aproject" do
    assert user_can_manage(:admin, api_fixture("groups")["aproject"])
  end

  test "owner can_manage aproject" do
    assert user_can_manage(:active, api_fixture("groups")["aproject"])
  end

  test "owner can_manage asubproject" do
    assert user_can_manage(:active, api_fixture("groups")["asubproject"])
  end

  test "viewer can't manage aproject" do
    refute user_can_manage(:project_viewer, api_fixture("groups")["aproject"])
  end

  test "viewer can't manage asubproject" do
    refute user_can_manage(:project_viewer, api_fixture("groups")["asubproject"])
  end

  test "subproject_admin can_manage asubproject" do
    assert user_can_manage(:subproject_admin, api_fixture("groups")["asubproject"])
  end

  test "detect ownership loop in project breadcrumbs" do
    # This test has an arbitrary time limit -- otherwise we'd just sit
    # here forever instead of reporting that the loop was not
    # detected. The test passes quickly, but fails slowly.
    Timeout::timeout 10 do
      get(:show,
          params: { id: api_fixture("groups")["project_owns_itself"]["uuid"] },
          session: session_for(:admin))
    end
    assert_response :success
  end

  test "project admin can remove collections from the project" do
    # Deleting an object that supports 'trash_at' should make it
    # completely inaccessible to API queries, not simply moved out of
    # the project.
    coll_key = "collection_to_remove_from_subproject"
    coll_uuid = api_fixture("collections")[coll_key]["uuid"]
    delete(:remove_item,
           params: { id: api_fixture("groups")["asubproject"]["uuid"],
             item_uuid: coll_uuid,
             format: "js" },
           session: session_for(:subproject_admin))
    assert_response :success
    assert_match(/\b#{coll_uuid}\b/, @response.body,
                 "removed object not named in response")

    use_token :subproject_admin
    assert_raise ArvadosApiClient::NotFoundException do
      Collection.find(coll_uuid, cache: false)
    end
  end

  test "project admin can remove items from project other than collections" do
    # An object which does not have an trash_at field (e.g. Specimen)
    # should be implicitly moved to the user's Home project when removed.
    specimen_uuid = api_fixture('specimens', 'in_asubproject')['uuid']
    delete(:remove_item,
           params: { id: api_fixture('groups', 'asubproject')['uuid'],
             item_uuid: specimen_uuid,
             format: 'js' },
           session: session_for(:subproject_admin))
    assert_response :success
    assert_match(/\b#{specimen_uuid}\b/, @response.body,
                 "removed object not named in response")

    use_token :subproject_admin
    new_specimen = Specimen.find(specimen_uuid)
    assert_equal api_fixture('users', 'subproject_admin')['uuid'], new_specimen.owner_uuid
  end

  test 'projects#show tab infinite scroll partial obeys limit' do
    get_contents_rows(limit: 1, filters: [['uuid','is_a',['arvados#job']]])
    assert_response :success
    assert_equal(1, json_response['content'].scan('<tr').count,
                 "Did not get exactly one row")
  end

  ['', ' asc', ' desc'].each do |direction|
    test "projects#show tab partial orders correctly by created_at#{direction}" do
      _test_tab_content_order direction
    end
  end

  def _test_tab_content_order direction
    get_contents_rows(limit: 100,
                      order: "created_at#{direction}",
                      filters: [['uuid','is_a',['arvados#job',
                                                'arvados#pipelineInstance']]])
    assert_response :success
    not_grouped_by_kind = nil
    last_timestamp = nil
    last_kind = nil
    found_kind = {}
    json_response['content'].scan /<tr[^>]+>/ do |tr_tag|
      found_timestamps = 0
      tr_tag.scan(/\ data-object-created-at=\"(.*?)\"/).each do |t,|
        if last_timestamp
          correct_operator = / desc$/ =~ direction ? :>= : :<=
          assert_operator(last_timestamp, correct_operator, t,
                          "Rows are not sorted by created_at#{direction}")
        end
        last_timestamp = t
        found_timestamps += 1
      end
      assert_equal(1, found_timestamps,
                   "Content row did not have exactly one timestamp")

      # Confirm that the test for timestamp ordering couldn't have
      # passed merely because the test fixtures have convenient
      # timestamps (e.g., there is only one pipeline and one job in
      # the project being tested, or there are no pipelines at all in
      # the project being tested):
      tr_tag.scan /\ data-kind=\"(.*?)\"/ do |kind|
        if last_kind and last_kind != kind and found_kind[kind]
          # We saw this kind before, then a different kind, then
          # this kind again. That means objects are not grouped by
          # kind.
          not_grouped_by_kind = true
        end
        found_kind[kind] ||= 0
        found_kind[kind] += 1
        last_kind = kind
      end
    end
    assert_equal(true, not_grouped_by_kind,
                 "Could not confirm that results are not grouped by kind")
  end

  def get_contents_rows params
    params = {
      id: api_fixture('users')['active']['uuid'],
      partial: :contents_rows,
      format: :json,
    }.merge(params)
    encoded_params = Hash[params.map { |k,v|
                            [k, (v.is_a?(Array) || v.is_a?(Hash)) ? v.to_json : v]
                          }]
    get :show, params: encoded_params, session: session_for(:active)
  end

  test "visit non-public project as anonymous when anonymous browsing is enabled and expect page not found" do
    Rails.configuration.Users.AnonymousUserToken = api_fixture('api_client_authorizations')['anonymous']['api_token']
    get(:show, params: {id: api_fixture('groups')['aproject']['uuid']})
    assert_response 404
    assert_match(/log ?in/i, @response.body)
  end

  test "visit home page as anonymous when anonymous browsing is enabled and expect login" do
    Rails.configuration.Users.AnonymousUserToken = api_fixture('api_client_authorizations')['anonymous']['api_token']
    get(:index)
    assert_response :redirect
    assert_match /\/users\/welcome/, @response.redirect_url
  end

  [
    nil,
    :active,
  ].each do |user|
    test "visit public projects page when anon config is enabled, as user #{user}, and expect page" do
      Rails.configuration.Users.AnonymousUserToken = api_fixture('api_client_authorizations')['anonymous']['api_token']

      if user
        get :public, params: {}, session: session_for(user)
      else
        get :public
      end

      assert_response :success
      assert_not_nil assigns(:objects)
      project_names = assigns(:objects).collect(&:name)
      assert_includes project_names, 'Unrestricted public data'
      assert_not_includes project_names, 'A Project'
      refute_empty css_select('[href="/projects/public"]')
    end
  end

  test "visit public projects page when anon config is not enabled as active user and expect 404" do
    Rails.configuration.Users.AnonymousUserToken = ""
    Rails.configuration.Workbench.EnablePublicProjectsPage = false
    get :public, params: {}, session: session_for(:active)
    assert_response 404
  end

  test "visit public projects page when anon config is enabled but public projects page is disabled as active user and expect 404" do
    Rails.configuration.Users.AnonymousUserToken = api_fixture('api_client_authorizations')['anonymous']['api_token']
    Rails.configuration.Workbench.EnablePublicProjectsPage = false
    get :public, params: {}, session: session_for(:active)
    assert_response 404
  end

  test "visit public projects page when anon config is not enabled as anonymous and expect login page" do
    Rails.configuration.Users.AnonymousUserToken = ""
    Rails.configuration.Workbench.EnablePublicProjectsPage = false
    get :public
    assert_response :redirect
    assert_match /\/users\/welcome/, @response.redirect_url
    assert_empty css_select('[href="/projects/public"]')
  end

  test "visit public projects page when anon config is enabled and public projects page is disabled and expect login page" do
    Rails.configuration.Users.AnonymousUserToken = api_fixture('api_client_authorizations')['anonymous']['api_token']
    Rails.configuration.Workbench.EnablePublicProjectsPage = false
    get :index
    assert_response :redirect
    assert_match /\/users\/welcome/, @response.redirect_url
    assert_empty css_select('[href="/projects/public"]')
  end

  test "visit public projects page when anon config is not enabled and public projects page is enabled and expect login page" do
    Rails.configuration.Workbench.EnablePublicProjectsPage = true
    get :index
    assert_response :redirect
    assert_match /\/users\/welcome/, @response.redirect_url
    assert_empty css_select('[href="/projects/public"]')
  end

  test "find a project and edit its description" do
    project = api_fixture('groups')['aproject']
    use_token :active
    found = Group.find(project['uuid'])
    found.description = 'test description update'
    found.save!
    get(:show, params: {id: project['uuid']}, session: session_for(:active))
    assert_includes @response.body, 'test description update'
  end

  test "find a project and edit description to textile description" do
    project = api_fixture('groups')['aproject']
    use_token :active
    found = Group.find(project['uuid'])
    found.description = '*test bold description for textile formatting*'
    found.save!
    get(:show, params: {id: project['uuid']}, session: session_for(:active))
    assert_includes @response.body, '<strong>test bold description for textile formatting</strong>'
  end

  test "find a project and edit description to html description" do
    project = api_fixture('groups')['aproject']
    use_token :active
    found = Group.find(project['uuid'])
    found.description = '<b>Textile</b> description with link to home page <a href="/">take me home</a>.'
    found.save!
    get(:show, params: {id: project['uuid']}, session: session_for(:active))
    assert_includes @response.body, '<b>Textile</b> description with link to home page <a href="/">take me home</a>.'
  end

  test "find a project and edit description to unsafe html description" do
    project = api_fixture('groups')['aproject']
    use_token :active
    found = Group.find(project['uuid'])
    found.description = 'Textile description with unsafe script tag <script language="javascript">alert("Hello there")</script>.'
    found.save!
    get(:show, params: {id: project['uuid']}, session: session_for(:active))
    assert_includes @response.body, 'Textile description with unsafe script tag alert("Hello there").'
  end

  # Tests #14519
  test "textile table on description renders as table html markup" do
    use_token :active
    project = api_fixture('groups')['aproject']
    textile_table = <<EOT
table(table table-striped table-condensed).
|_. First Header |_. Second Header |
|Content Cell |Content Cell |
|Content Cell |Content Cell |
EOT
    found = Group.find(project['uuid'])
    found.description = textile_table
    found.save!
    get(:show, params: {id: project['uuid']}, session: session_for(:active))
    assert_includes @response.body, '<th>First Header'
    assert_includes @response.body, '<td>Content Cell'
  end

  test "find a project and edit description to textile description with link to object" do
    project = api_fixture('groups')['aproject']
    use_token :active
    found = Group.find(project['uuid'])

    # uses 'Link to object' as a hyperlink for the object
    found.description = '"Link to object":' + api_fixture('groups')['asubproject']['uuid']
    found.save!
    get(:show, params: {id: project['uuid']}, session: session_for(:active))

    # check that input was converted to textile, not staying as inputted
    refute_includes  @response.body,'"Link to object"'
    refute_empty css_select('[href="/groups/zzzzz-j7d0g-axqo7eu9pwvna1x"]')
  end

  test "project viewer can't see project sharing tab" do
    project = api_fixture('groups')['aproject']
    get(:show, params: {id: project['uuid']}, session: session_for(:project_viewer))
    refute_includes @response.body, '<div id="Sharing"'
    assert_includes @response.body, '<div id="Data_collections"'
  end

  [
    'admin',
    'active',
  ].each do |username|
    test "#{username} can see project sharing tab" do
     project = api_fixture('groups')['aproject']
     get(:show, params: {id: project['uuid']}, session: session_for(username))
     assert_includes @response.body, '<div id="Sharing"'
     assert_includes @response.body, '<div id="Data_collections"'
    end
  end

  [
    ['admin',true],
    ['active',true],
    ['project_viewer',false],
  ].each do |user, can_move|
    test "#{user} can move subproject from project #{can_move}" do
      get(:show, params: {id: api_fixture('groups')['aproject']['uuid']}, session: session_for(user))
      if can_move
        assert_includes @response.body, 'Move project...'
      else
        refute_includes @response.body, 'Move project...'
      end
    end
  end

  [:admin, :active].each do |user|
    test "in dashboard other index page links as #{user}" do
      get :index, params: {}, session: session_for(user)

      [["processes", "/all_processes"],
       ["collections", "/collections"],
      ].each do |target, path|
        assert_includes @response.body, "href=\"#{path}\""
        assert_includes @response.body, "All #{target}"
      end
    end
  end

  test "dashboard should show the correct status for processes" do
    get :index, params: {}, session: session_for(:active)
    assert_select 'div.panel-body.recent-processes' do
      [
        {
          fixture: 'container_requests',
          state: 'completed',
          selectors: [['div.progress', false],
                      ['span.label.label-success', true, 'Complete']]
        },
        {
          fixture: 'container_requests',
          state: 'uncommitted',
          selectors: [['div.progress', false],
                      ['span.label.label-default', true, 'Uncommitted']]
        },
        {
          fixture: 'container_requests',
          state: 'queued',
          selectors: [['div.progress', false],
                      ['span.label.label-default', true, 'Queued']]
        },
        {
          fixture: 'container_requests',
          state: 'running',
          selectors: [['.label-info', true, 'Running']]
        },
        {
          fixture: 'pipeline_instances',
          state: 'new_pipeline',
          selectors: [['div.progress', false],
                      ['span.label.label-default', true, 'Not started']]
        },
        {
          fixture: 'pipeline_instances',
          state: 'pipeline_in_running_state',
          selectors: [['.label-info', true, 'Running']]
        },
      ].each do |c|
        uuid = api_fixture(c[:fixture])[c[:state]]['uuid']
        assert_select "div.dashboard-panel-info-row.row-#{uuid}" do
          if c.include? :selectors
            c[:selectors].each do |selector, should_show, label|
              assert_select selector, should_show, "UUID #{uuid} should #{should_show ? '' : 'not'} show '#{selector}'"
              if should_show and not label.nil?
                assert_select selector, label, "UUID #{uuid} state label should show #{label}"
              end
            end
          end
        end
      end
    end
  end

  test "visit a public project and verify the public projects page link exists" do
    Rails.configuration.Users.AnonymousUserToken = api_fixture('api_client_authorizations')['anonymous']['api_token']
    uuid = api_fixture('groups')['anonymously_accessible_project']['uuid']
    get :show, params: {id: uuid}
    project = assigns(:object)
    assert_equal uuid, project['uuid']
    refute_empty css_select("[href=\"/projects/#{project['uuid']}\"]")
    assert_includes @response.body, "<a href=\"/projects/public\">Public Projects</a>"
  end

  test 'all_projects unaffected by params after use by ProjectsController (#6640)' do
    @controller = ProjectsController.new
    project_uuid = api_fixture('groups')['aproject']['uuid']
    get :index, params: {
      filters: [['uuid', '<', project_uuid]].to_json,
      limit: 0,
      offset: 1000,
    }, session: session_for(:active)
    assert_select "#projects-menu + ul li.divider ~ li a[href=\"/projects/#{project_uuid}\"]"
  end

  [
    ["active", 5, ["aproject", "asubproject"], "anonymously_accessible_project"],
    ["user1_with_load", 2, ["project_with_10_collections"], "project_with_2_pipelines_and_60_crs"],
    ["admin", 5, ["anonymously_accessible_project", "subproject_in_anonymous_accessible_project"], "aproject"],
  ].each do |user, page_size, tree_segment, unexpected|
    # Note: this test is sensitive to database collation. It passes
    # with en_US.UTF-8.
    test "build my projects tree for #{user} user and verify #{unexpected} is omitted" do
      use_token user

      tree, _, _ = @controller.send(:my_wanted_projects_tree,
                                    User.current,
                                    page_size)

      tree_segment_at_depth_1 = api_fixture('groups')[tree_segment[0]]
      tree_segment_at_depth_2 = api_fixture('groups')[tree_segment[1]] if tree_segment[1]

      node_depth = {}
      tree.each do |x|
        node_depth[x[:object]['uuid']] = x[:depth]
      end

      assert_equal(1, node_depth[tree_segment_at_depth_1['uuid']])
      assert_equal(2, node_depth[tree_segment_at_depth_2['uuid']]) if tree_segment[1]

      unexpected_project = api_fixture('groups')[unexpected]
      assert_nil(node_depth[unexpected_project['uuid']], node_depth.inspect)
    end
  end

  [
    ["active", 1],
    ["project_viewer", 1],
    ["admin", 0],
  ].each do |user, size|
    test "starred projects for #{user}" do
      use_token user
      ctrl = ProjectsController.new
      current_user = User.find(api_fixture('users')[user]['uuid'])
      my_starred_project = ctrl.send :my_starred_projects, current_user, ''
      assert_equal(size, my_starred_project.andand.size)

      ctrl2 = ProjectsController.new
      current_user = User.find(api_fixture('users')[user]['uuid'])
      my_starred_project = ctrl2.send :my_starred_projects, current_user, ''
      assert_equal(size, my_starred_project.andand.size)
    end
  end

  test "unshare project and verify that it is no longer included in shared user's starred projects" do
    # remove sharing link
    use_token :system_user
    Link.find(api_fixture('links')['share_starred_project_with_project_viewer']['uuid']).destroy

    # verify that project is no longer included in starred projects
    use_token :project_viewer
    current_user = User.find(api_fixture('users')['project_viewer']['uuid'])
    ctrl = ProjectsController.new
    my_starred_project = ctrl.send :my_starred_projects, current_user, ''
    assert_equal(0, my_starred_project.andand.size)

    # share it again
    @controller = LinksController.new
    post :create, params: {
      link: {
        link_class: 'permission',
        name: 'can_read',
        head_uuid: api_fixture('groups')['starred_and_shared_active_user_project']['uuid'],
        tail_uuid: api_fixture('users')['project_viewer']['uuid'],
      },
      format: :json
    }, session: session_for(:system_user)

    # verify that the project is again included in starred projects
    use_token :project_viewer
    ctrl = ProjectsController.new
    my_starred_project = ctrl.send :my_starred_projects, current_user, ''
    assert_equal(1, my_starred_project.andand.size)
  end
end
