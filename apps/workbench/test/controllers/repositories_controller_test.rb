require 'test_helper'
require 'helpers/share_object_helper'

class RepositoriesControllerTest < ActionController::TestCase
  include ShareObjectHelper

  [
    :active, #owner
    :admin,
  ].each do |user|
    test "#{user} shares repository with a user and group" do
      uuid_list = [api_fixture("groups")["future_project_viewing_group"]["uuid"],
                   api_fixture("users")["future_project_user"]["uuid"]]
      post(:share_with, {
             id: api_fixture("repositories")["foo"]["uuid"],
             uuids: uuid_list,
             format: "json"},
           session_for(user))
      assert_response :success
      assert_equal(uuid_list, json_response["success"])
    end
  end

  test "user with repository read permission cannot add permissions" do
    share_uuid = api_fixture("users")["project_viewer"]["uuid"]
    post(:share_with, {
           id: api_fixture("repositories")["arvados"]["uuid"],
           uuids: [share_uuid],
           format: "json"},
         session_for(:spectator))
    assert_response 422
    assert(json_response["errors"].andand.
             any? { |msg| msg.start_with?("#{share_uuid}: ") },
           "JSON response missing properly formatted sharing error")
  end

  test "admin can_manage repository" do
    assert user_can_manage(:admin, api_fixture("repositories")["foo"])
  end

  test "owner can_manage repository" do
    assert user_can_manage(:active, api_fixture("repositories")["foo"])
  end

  test "viewer cannot manage repository" do
    refute user_can_manage(:spectator, api_fixture("repositories")["arvados"])
  end

  [
    [:active, ['#Sharing', '#Advanced']],
    [:admin,  ['#Attributes', '#Sharing', '#Advanced']],
  ].each do |user, panes|
    test "#{user} sees panes #{panes}" do
      get :show, {
        id: api_fixture('repositories')['foo']['uuid']
      }, session_for(user)
      assert_response :success

      panes = css_select('[data-toggle=tab]').select do |pane|
        pane_name = pane.attributes['href']
        assert_equal true, (panes.include? pane_name),
                     "Did not find pane #{pane_name}"
      end
    end
  end
end
