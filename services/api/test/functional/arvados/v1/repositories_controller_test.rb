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

  test "get_all_permissions takes into account is_active flag" do
    r = nil
    act_as_user users(:active) do
      r = Repository.create! name: 'active/testrepo'
    end
    act_as_system_user do
      u = users(:active)
      u.is_active = false
      u.save!
    end
    authorize_with :admin
    get :get_all_permissions
    assert_response :success
    json_response['repositories'].each do |r|
      r['user_permissions'].each do |user_uuid, perms|
        refute_equal user_uuid, users(:active).uuid
      end
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

  test "get_all_permissions lists all repos regardless of permissions" do
    act_as_system_user do
      # Create repos that could potentially be left out of the
      # permission list by accident.

      # No authorized_key, no username (this can't even be done
      # without skipping validations)
      r = Repository.create name: 'root/testrepo'
      assert r.save validate: false

      r = Repository.create name: 'invalid username / repo name', owner_uuid: users(:inactive).uuid
      assert r.save validate: false
    end
    authorize_with :admin
    get :get_all_permissions
    assert_response :success
    assert_equal(Repository.count, json_response["repositories"].size)
  end

  test "get_all_permissions lists user permissions for users with no authorized keys" do
    authorize_with :admin
    AuthorizedKey.destroy_all
    get :get_all_permissions
    assert_response :success
    assert_equal(Repository.count, json_response["repositories"].size)
    repos_with_perms = []
    json_response['repositories'].each do |repo|
      if repo['user_permissions'].any?
        repos_with_perms << repo['uuid']
      end
    end
    assert_not_empty repos_with_perms, 'permissions are missing'
  end

  # Ensure get_all_permissions correctly describes what the normal
  # permission system would do.
  test "get_all_permissions obeys group permissions" do
    act_as_user system_user do
      r = Repository.create!(name: 'admin/groupcanwrite', owner_uuid: users(:admin).uuid)
      g = Group.create!(group_class: 'group', name: 'repo-writers')
      u1 = users(:active)
      u2 = users(:spectator)
      Link.create!(tail_uuid: g.uuid, head_uuid: r.uuid, link_class: 'permission', name: 'can_manage')
      Link.create!(tail_uuid: u1.uuid, head_uuid: g.uuid, link_class: 'permission', name: 'can_write')
      Link.create!(tail_uuid: u2.uuid, head_uuid: g.uuid, link_class: 'permission', name: 'can_read')

      r = Repository.create!(name: 'admin/groupreadonly', owner_uuid: users(:admin).uuid)
      g = Group.create!(group_class: 'group', name: 'repo-readers')
      u1 = users(:active)
      u2 = users(:spectator)
      Link.create!(tail_uuid: g.uuid, head_uuid: r.uuid, link_class: 'permission', name: 'can_read')
      Link.create!(tail_uuid: u1.uuid, head_uuid: g.uuid, link_class: 'permission', name: 'can_write')
      Link.create!(tail_uuid: u2.uuid, head_uuid: g.uuid, link_class: 'permission', name: 'can_read')
    end
    authorize_with :admin
    get :get_all_permissions
    assert_response :success
    json_response['repositories'].each do |repo|
      repo['user_permissions'].each do |user_uuid, perms|
        u = User.find_by_uuid(user_uuid)
        if perms['can_read']
          assert u.can? read: repo['uuid']
          assert_match /R/, perms['gitolite_permissions']
        else
          refute_match /R/, perms['gitolite_permissions']
        end
        if perms['can_write']
          assert u.can? write: repo['uuid']
          assert_match /RW/, perms['gitolite_permissions']
        else
          refute_match /W/, perms['gitolite_permissions']
        end
        if perms['can_manage']
          assert u.can? manage: repo['uuid']
          assert_match /RW/, perms['gitolite_permissions']
        end
      end
    end
  end

  test "default index includes fetch_url" do
    authorize_with :active
    get(:index)
    assert_response :success
    assert_includes(json_response["items"].map { |r| r["fetch_url"] },
                    "git@git.zzzzz.arvadosapi.com:active/foo.git")
  end

  [
    {cfg: :git_repo_ssh_base, cfgval: "git@example.com:", match: %r"^git@example.com:/"},
    {cfg: :git_repo_ssh_base, cfgval: true, match: %r"^git@git.zzzzz.arvadosapi.com:/"},
    {cfg: :git_repo_ssh_base, cfgval: false, refute: /^git@/ },
    {cfg: :git_repo_https_base, cfgval: "https://example.com/", match: %r"https://example.com/"},
    {cfg: :git_repo_https_base, cfgval: true, match: %r"^https://git.zzzzz.arvadosapi.com/"},
    {cfg: :git_repo_https_base, cfgval: false, refute: /^http/ },
  ].each do |expect|
    test "set #{expect[:cfg]} to #{expect[:cfgval]}" do
      Rails.configuration.send expect[:cfg].to_s+"=", expect[:cfgval]
      authorize_with :active
      get :index
      assert_response :success
      json_response['items'].each do |r|
        if expect[:refute]
          r['clone_urls'].each do |u|
            refute_match expect[:refute], u
          end
        else
          assert r['clone_urls'].any? do |u|
            expect[:prefix].match u
          end
        end
      end
    end
  end

  test "select push_url in index" do
    authorize_with :active
    get(:index, {select: ["uuid", "push_url"]})
    assert_response :success
    assert_includes(json_response["items"].map { |r| r["push_url"] },
                    "git@git.zzzzz.arvadosapi.com:active/foo.git")
  end

  test "select clone_urls in index" do
    authorize_with :active
    get(:index, {select: ["uuid", "clone_urls"]})
    assert_response :success
    assert_includes(json_response["items"].map { |r| r["clone_urls"] }.flatten,
                    "git@git.zzzzz.arvadosapi.com:active/foo.git")
  end
end
