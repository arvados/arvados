require 'test_helper'

class ProjectsControllerTest < ActionController::TestCase
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
        el.attributes['href'].match /project.*owner_uuid.*#{readonly_project_uuid}/
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

  def user_can_manage(user_sym, group_key)
    get(:show, {id: api_fixture("groups")[group_key]["uuid"]},
        session_for(user_sym))
    is_manager = assigns(:user_is_manager)
    assert_not_nil(is_manager, "user_is_manager flag not set")
    if not is_manager
      assert_empty(assigns(:share_links),
                   "non-manager has share links set")
    end
    is_manager
  end

  test "admin can_manage aproject" do
    assert user_can_manage(:admin, "aproject")
  end

  test "owner can_manage aproject" do
    assert user_can_manage(:active, "aproject")
  end

  test "owner can_manage asubproject" do
    assert user_can_manage(:active, "asubproject")
  end

  test "viewer can't manage aproject" do
    refute user_can_manage(:project_viewer, "aproject")
  end

  test "viewer can't manage asubproject" do
    refute user_can_manage(:project_viewer, "asubproject")
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
end
