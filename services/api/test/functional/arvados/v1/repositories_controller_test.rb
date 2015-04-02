require 'test_helper'

class Arvados::V1::RepositoriesControllerTest < ActionController::TestCase
  test "should get_all_logins with admin token" do
    authorize_with :admin
    get :get_all_permissions
    assert_response :success
  end

  test "should get_all_logins with non-admin token" do
    authorize_with :active
    get :get_all_permissions
    assert_response 403
  end

  test "get_all_permissions gives RW to repository owner" do
    authorize_with :admin
    get :get_all_permissions
    assert_response :success
    ok = false
    json_response['repositories'].each do |repo|
      if repo['uuid'] == repositories(:repository2).uuid
        if repo['user_permissions'][users(:active).uuid]['can_write']
          ok = true
        end
      end
    end
    assert_equal(true, ok,
                 "No permission on own repo '@{repositories(:repository2).uuid}'")
  end

  test "get_all_permissions takes into account is_admin flag" do
    authorize_with :admin
    get :get_all_permissions
    assert_response :success
    json_response['repositories'].each do |repo|
      assert_not_nil(repo['user_permissions'][users(:admin).uuid],
                     "Admin user is not listed in perms for #{repo['uuid']}")
      assert_equal(true,
                   repo['user_permissions'][users(:admin).uuid]['can_write'],
                   "Admin has no perms for #{repo['uuid']}")
    end
  end

  test "get_all_permissions does not give any access to user without permission" do
    viewer_uuid = users(:project_viewer).uuid
    assert_equal(authorized_keys(:project_viewer).authorized_user_uuid,
                 viewer_uuid,
                 "project_viewer must have an authorized_key for this test to work")
    authorize_with :admin
    get :get_all_permissions
    assert_response :success
    readable_repos = json_response["repositories"].select do |repo|
      repo["user_permissions"].has_key?(viewer_uuid)
    end
    assert_equal(["arvados"], readable_repos.map { |r| r["name"] },
                 "project_viewer should only have permissions on public repos")
  end

  test "get_all_permissions gives gitolite R to user with read-only access" do
    authorize_with :admin
    get :get_all_permissions
    assert_response :success
    found_it = false
    assert_equal(authorized_keys(:spectator).authorized_user_uuid,
                 users(:spectator).uuid,
                 "spectator must have an authorized_key for this test to work")
    json_response['repositories'].each do |repo|
      next unless repo['uuid'] == repositories(:foo).uuid
      assert_equal('R',
                   repo['user_permissions'][users(:spectator).uuid]['gitolite_permissions'],
                   "spectator user should have just R access to #{repo['uuid']}")
      found_it = true
    end
    assert_equal true, found_it, "spectator user does not have R on foo repo"
  end

  test "get_all_permissions provides admin and active user keys" do
    authorize_with :admin
    get :get_all_permissions
    assert_response :success
    [:active, :admin].each do |u|
      assert_equal(1, json_response['user_keys'][users(u).uuid].andand.count,
                   "expected 1 key for #{u} (#{users(u).uuid})")
      assert_equal(json_response['user_keys'][users(u).uuid][0]['public_key'],
                   authorized_keys(u).public_key,
                   "response public_key does not match fixture #{u}.")
    end
  end

  test "default index includes fetch_url" do
    authorize_with :active
    get(:index)
    assert_response :success
    assert_includes(json_response["items"].map { |r| r["fetch_url"] },
                    "git@git.zzzzz.arvadosapi.com:active/foo.git")
  end

  test "setting git_host changes fetch_url" do
    Rails.configuration.git_host = "example.com"
    authorize_with :active
    get(:index)
    assert_response :success
    assert_includes(json_response["items"].map { |r| r["fetch_url"] },
                    "git@example.com:active/foo.git")
  end

  test "can select push_url in index" do
    authorize_with :active
    get(:index, {select: ["uuid", "push_url"]})
    assert_response :success
    assert_includes(json_response["items"].map { |r| r["push_url"] },
                    "git@git.zzzzz.arvadosapi.com:active/foo.git")
  end
end
