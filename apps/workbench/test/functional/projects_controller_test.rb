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
end
