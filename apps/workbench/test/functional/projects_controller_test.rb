require 'test_helper'

class ProjectsControllerTest < ActionController::TestCase
  test "inactive user is asked to sign user agreements on front page" do
    get :index, {}, session_for(:inactive)
    assert_response :success
    assert_not_empty assigns(:required_user_agreements),
    "Inactive user did not have required_user_agreements"
    assert_template 'user_agreements/index',
    "Inactive user was not presented with a user agreement at the front page"
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
    json_response = Oj.load(@response.body)
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
    json_response = Oj.load(@response.body)
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
end
