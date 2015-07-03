require 'test_helper'
require 'helpers/share_object_helper'

class ProjectsControllerTest < ActionController::TestCase
  include ShareObjectHelper

  test "invited user is asked to sign user agreements on front page" do
    get :index, {}, session_for(:inactive)
    assert_response :redirect
    assert_match(/^#{Regexp.escape(user_agreements_url)}\b/,
                 @response.redirect_url,
                 "Inactive user was not redirected to user_agreements page")
  end

  test "uninvited user is asked to wait for activation" do
    get :index, {}, session_for(:inactive_uninvited)
    assert_response :redirect
    assert_match(/^#{Regexp.escape(inactive_users_url)}\b/,
                 @response.redirect_url,
                 "Uninvited user was not redirected to inactive user page")
  end

  [[:active, true],
   [:project_viewer, false]].each do |which_user, should_show|
    test "create subproject button #{'not ' unless should_show} shown to #{which_user}" do
      readonly_project_uuid = api_fixture('groups')['aproject']['uuid']
      get :show, {
        id: readonly_project_uuid
      }, session_for(which_user)
      buttons = css_select('[data-method=post]').select do |el|
        el.attributes['data-remote-href'].match /project.*owner_uuid.*#{readonly_project_uuid}/
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
    post(:share_with, {
           id: api_fixture("groups")["asubproject"]["uuid"],
           uuids: uuid_list,
           format: "json"},
         session_for(:active))
    assert_response :success
    assert_equal(uuid_list, json_response["success"])
  end

  test "user with project read permission can't add permissions" do
    share_uuid = api_fixture("users")["spectator"]["uuid"]
    post(:share_with, {
           id: api_fixture("groups")["aproject"]["uuid"],
           uuids: [share_uuid],
           format: "json"},
         session_for(:project_viewer))
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
          { id: api_fixture("groups")["project_owns_itself"]["uuid"] },
          session_for(:admin))
    end
    assert_response :success
  end

  test "project admin can remove collections from the project" do
    # Deleting an object that supports 'expires_at' should make it
    # completely inaccessible to API queries, not simply moved out of the project.
    coll_key = "collection_to_remove_from_subproject"
    coll_uuid = api_fixture("collections")[coll_key]["uuid"]
    delete(:remove_item,
           { id: api_fixture("groups")["asubproject"]["uuid"],
             item_uuid: coll_uuid,
             format: "js" },
           session_for(:subproject_admin))
    assert_response :success
    assert_match(/\b#{coll_uuid}\b/, @response.body,
                 "removed object not named in response")

    use_token :subproject_admin
    assert_raise ArvadosApiClient::NotFoundException do
      Collection.find(coll_uuid)
    end
  end

  test "project admin can remove items from project other than collections" do
    # An object which does not have an expired_at field (e.g. Specimen)
    # should be implicitly moved to the user's Home project when removed.
    specimen_uuid = api_fixture('specimens', 'in_asubproject')['uuid']
    delete(:remove_item,
           { id: api_fixture('groups', 'asubproject')['uuid'],
             item_uuid: specimen_uuid,
             format: 'js' },
           session_for(:subproject_admin))
    assert_response :success
    assert_match(/\b#{specimen_uuid}\b/, @response.body,
                 "removed object not named in response")

    use_token :subproject_admin
    new_specimen = Specimen.find(specimen_uuid)
    assert_equal api_fixture('users', 'subproject_admin')['uuid'], new_specimen.owner_uuid
  end

  # An object which does not offer an expired_at field but has a xx_owner_uuid_name_unique constraint
  # will be renamed when removed and another object with the same name exists in user's home project.
  [
    ['groups', 'subproject_in_asubproject_with_same_name_as_one_in_active_user_home'],
    ['pipeline_templates', 'template_in_asubproject_with_same_name_as_one_in_active_user_home'],
  ].each do |dm, fixture|
    test "removing #{dm} from a subproject results in renaming it when there is another such object with same name in home project" do
      object = api_fixture(dm, fixture)
      delete(:remove_item,
             { id: api_fixture('groups', 'asubproject')['uuid'],
               item_uuid: object['uuid'],
               format: 'js' },
             session_for(:active))
      assert_response :success
      assert_match(/\b#{object['uuid']}\b/, @response.body,
                   "removed object not named in response")
      use_token :active
      if dm.eql?('groups')
        found = Group.find(object['uuid'])
      else
        found = PipelineTemplate.find(object['uuid'])
      end
      assert_equal api_fixture('users', 'active')['uuid'], found.owner_uuid
      assert_equal true, found.name.include?(object['name'] + ' removed from ')
    end
  end

  test 'projects#show tab infinite scroll partial obeys limit' do
    get_contents_rows(limit: 1, filters: [['uuid','is_a',['arvados#job']]])
    assert_response :success
    assert_equal(1, json_response['content'].scan('<tr').count,
                 "Did not get exactly one row")
  end

  ['', ' asc', ' desc'].each do |direction|
    test "projects#show tab partial orders correctly by #{direction}" do
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
    get :show, encoded_params, session_for(:active)
  end

  test "visit non-public project as anonymous when anonymous browsing is enabled and expect page not found" do
    Rails.configuration.anonymous_user_token = api_fixture('api_client_authorizations')['anonymous']['api_token']
    get(:show, {id: api_fixture('groups')['aproject']['uuid']})
    assert_response 404
    assert_includes @response.inspect, 'you are not logged in'
  end

  test "visit home page as anonymous when anonymous browsing is enabled and expect login" do
    Rails.configuration.anonymous_user_token = api_fixture('api_client_authorizations')['anonymous']['api_token']
    get(:index)
    assert_response :redirect
    assert_match /\/users\/welcome/, @response.redirect_url
  end

  [
    nil,
    :active,
  ].each do |user|
    test "visit public projects page when anon config is enabled, as user #{user}, and expect page" do
      Rails.configuration.anonymous_user_token = api_fixture('api_client_authorizations')['anonymous']['api_token']

      if user
        get :public, {}, session_for(user)
      else
        get :public
      end

      assert_response :success
      assert_not_nil assigns(:objects)
      project_names = assigns(:objects).collect(&:name)
      assert_includes project_names, 'Unrestricted public data'
      assert_not_includes project_names, 'A Project'
    end
  end

  test "visit public projects page when anon config is not enabled as active user and expect 404" do
    get :public, {}, session_for(:active)
    assert_response 404
  end

  test "visit public projects page when anon config is enabled but public projects page is disabled and expect 404" do
    Rails.configuration.anonymous_user_token = api_fixture('api_client_authorizations')['anonymous']['api_token']
    Rails.configuration.enable_public_projects_page = false
    get :public, {}, session_for(:active)
    assert_response 404
  end

  test "visit public projects page when anon config is not enabled as anonymous and expect login page" do
    get :public
    assert_response :redirect
    assert_match /\/users\/welcome/, @response.redirect_url
  end
end
