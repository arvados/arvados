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
end
