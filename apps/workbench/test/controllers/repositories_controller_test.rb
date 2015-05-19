require 'test_helper'
require 'helpers/repository_stub_helper'
require 'helpers/share_object_helper'

class RepositoriesControllerTest < ActionController::TestCase
  include RepositoryStubHelper
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
  ].each do |user, expected_panes|
    test "#{user} sees panes #{expected_panes}" do
      get :show, {
        id: api_fixture('repositories')['foo']['uuid']
      }, session_for(user)
      assert_response :success

      panes = css_select('[data-toggle=tab]').each do |pane|
        pane_name = pane.attributes['href']
        assert_includes expected_panes, pane_name
      end
    end
  end

  ### Browse repository content

  [:active, :spectator].each do |user|
    test "show tree to #{user}" do
      reset_api_fixtures_after_test false
      sha1, _, _ = stub_repo_content
      get :show_tree, {
        id: api_fixture('repositories')['foo']['uuid'],
        commit: sha1,
      }, session_for(user)
      assert_response :success
      assert_select 'tr td a', 'COPYING'
      assert_select 'tr td', '625 bytes'
      assert_select 'tr td a', 'apps'
      assert_select 'tr td a', 'workbench'
      assert_select 'tr td a', 'Gemfile'
      assert_select 'tr td', '33.7 KiB'
    end

    test "show commit to #{user}" do
      reset_api_fixtures_after_test false
      sha1, commit, _ = stub_repo_content
      get :show_commit, {
        id: api_fixture('repositories')['foo']['uuid'],
        commit: sha1,
      }, session_for(user)
      assert_response :success
      assert_select 'pre', h(commit)
    end

    test "show blob to #{user}" do
      reset_api_fixtures_after_test false
      sha1, _, filedata = stub_repo_content filename: 'COPYING'
      get :show_blob, {
        id: api_fixture('repositories')['foo']['uuid'],
        commit: sha1,
        path: 'COPYING',
      }, session_for(user)
      assert_response :success
      assert_select 'pre', h(filedata)
    end
  end

  ['', '/'].each do |path|
    test "show tree with path '#{path}'" do
      reset_api_fixtures_after_test false
      sha1, _, _ = stub_repo_content filename: 'COPYING'
      get :show_tree, {
        id: api_fixture('repositories')['foo']['uuid'],
        commit: sha1,
        path: path,
      }, session_for(:active)
      assert_response :success
      assert_select 'tr td', 'COPYING'
    end
  end
end
