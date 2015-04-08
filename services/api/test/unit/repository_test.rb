require 'test_helper'
require 'helpers/git_test_helper'

class RepositoryTest < ActiveSupport::TestCase
  include GitTestHelper

  def new_repo(owner_key, attrs={})
    set_user_from_auth owner_key
    owner = users(owner_key)
    Repository.new({owner_uuid: owner.uuid}.merge(attrs))
  end

  def changed_repo(repo_key, changes)
    repo = repositories(repo_key)
    changes.each_pair { |attr, value| repo.send("#{attr}=".to_sym, value) }
    repo
  end

  def default_git_url(repo_name, user_name=nil)
    if user_name
      "git@git.%s.arvadosapi.com:%s/%s.git" %
        [Rails.configuration.uuid_prefix, user_name, repo_name]
    else
      "git@git.%s.arvadosapi.com:%s.git" %
        [Rails.configuration.uuid_prefix, repo_name]
    end
  end

  def assert_server_path(path_tail, repo_sym)
    assert_equal(File.join(Rails.configuration.git_repositories_dir, path_tail),
                 repositories(repo_sym).server_path)
  end

  ### name validation

  {active: "active/", admin: "admin/", system_user: ""}.
      each_pair do |user_sym, name_prefix|
    %w(a aa a0 aA Aa AA A0).each do |name|
      test "'#{name_prefix}#{name}' is a valid name for #{user_sym} repo" do
        repo = new_repo(user_sym, name: name_prefix + name)
        assert(repo.valid?)
      end
    end

    test "name is required for #{user_sym} repo" do
      refute(new_repo(user_sym).valid?)
    end

    test "repo name beginning with numeral is invalid for #{user_sym}" do
      repo = new_repo(user_sym, name: "#{name_prefix}0a")
      refute(repo.valid?)
    end

    "\\.-_/!@#$%^&*()[]{}".each_char do |bad_char|
      test "name containing #{bad_char.inspect} is invalid for #{user_sym}" do
        repo = new_repo(user_sym, name: "#{name_prefix}bad#{bad_char}reponame")
        refute(repo.valid?)
      end
    end
  end

  test "admin can create valid repo for other user with correct name prefix" do
    owner = users(:active)
    repo = new_repo(:admin, name: "#{owner.username}/validnametest",
                    owner_uuid: owner.uuid)
    assert(repo.valid?)
  end

  test "admin can create valid system repo without name prefix" do
    repo = new_repo(:admin, name: "validnametest",
                    owner_uuid: users(:system_user).uuid)
    assert(repo.valid?)
  end

  test "repo name prefix must match owner_uuid username" do
    repo = new_repo(:admin, name: "admin/badusernametest",
                    owner_uuid: users(:active).uuid)
    refute(repo.valid?)
  end

  test "repo name prefix must be empty for system repo" do
    repo = new_repo(:admin, name: "root/badprefixtest",
                    owner_uuid: users(:system_user).uuid)
    refute(repo.valid?)
  end

  ### owner validation

  test "name must be unique per user" do
    repo = new_repo(:active, name: repositories(:foo).name)
    refute(repo.valid?)
  end

  test "name can be duplicated across users" do
    repo = new_repo(:active, name: "active/#{repositories(:arvados).name}")
    assert(repo.valid?)
  end

  test "repository cannot be owned by a group" do
    set_user_from_auth :active
    repo = Repository.new(owner_uuid: groups(:all_users).uuid,
                          name: "ownedbygroup")
    refute(repo.valid?)
    refute_empty(repo.errors[:owner_uuid] || [])
  end

  ### URL generation

  test "fetch_url" do
    repo = new_repo(:active, name: "active/fetchtest")
    repo.save
    assert_equal(default_git_url("fetchtest", "active"), repo.fetch_url)
  end

  test "fetch_url owned by system user" do
    set_user_from_auth :admin
    repo = Repository.new(owner_uuid: users(:system_user).uuid,
                          name: "fetchtest")
    repo.save
    assert_equal(default_git_url("fetchtest"), repo.fetch_url)
  end

  test "push_url" do
    repo = new_repo(:active, name: "active/pushtest")
    repo.save
    assert_equal(default_git_url("pushtest", "active"), repo.push_url)
  end

  test "push_url owned by system user" do
    set_user_from_auth :admin
    repo = Repository.new(owner_uuid: users(:system_user).uuid,
                          name: "pushtest")
    repo.save
    assert_equal(default_git_url("pushtest"), repo.push_url)
  end

  ### Path generation

  test "disk path stored by UUID" do
    assert_server_path("zzzzz-s0uqq-382brsig8rp3666/.git", :foo)
  end

  test "disk path stored by name" do
    assert_server_path("arvados/.git", :arvados)
  end

  test "disk path for repository not on disk" do
    assert_nil(Repository.new.server_path)
  end

  ### Repository creation

  test "non-admin can create a repository for themselves" do
    repo = new_repo(:active, name: "active/newtestrepo")
    assert(repo.save)
  end

  test "non-admin can't create a repository for another visible user" do
    repo = new_repo(:active, name: "repoforanon",
                    owner_uuid: users(:anonymous).uuid)
    assert_not_allowed { repo.save }
  end

  test "admin can create a repository for themselves" do
    repo = new_repo(:admin, name: "admin/newtestrepo")
    assert(repo.save)
  end

  test "admin can create a repository for others" do
    repo = new_repo(:admin, name: "active/repoforactive",
                    owner_uuid: users(:active).uuid)
    assert(repo.save)
  end

  test "admin can create a system repository" do
    repo = new_repo(:admin, name: "repoforsystem",
                    owner_uuid: users(:system_user).uuid)
    assert(repo.save)
  end

  ### Repository destruction

  test "non-admin can destroy their own repository" do
    set_user_from_auth :active
    assert(repositories(:foo).destroy)
  end

  test "non-admin can't destroy others' repository" do
    set_user_from_auth :active
    assert_not_allowed { repositories(:repository3).destroy }
  end

  test "non-admin can't destroy system repository" do
    set_user_from_auth :active
    assert_not_allowed { repositories(:arvados).destroy }
  end

  test "admin can destroy their own repository" do
    set_user_from_auth :admin
    assert(repositories(:repository3).destroy)
  end

  test "admin can destroy others' repository" do
    set_user_from_auth :admin
    assert(repositories(:foo).destroy)
  end

  test "admin can destroy system repository" do
    set_user_from_auth :admin
    assert(repositories(:arvados).destroy)
  end

  ### Changing ownership

  test "non-admin can't make their repository a system repository" do
    set_user_from_auth :active
    repo = changed_repo(:foo, owner_uuid: users(:system_user).uuid)
    assert_not_allowed { repo.save }
  end

  test "admin can give their repository to someone else" do
    set_user_from_auth :admin
    repo = changed_repo(:repository3, owner_uuid: users(:active).uuid,
                        name: "active/foo3")
    assert(repo.save)
  end

  test "admin can make their repository a system repository" do
    set_user_from_auth :admin
    repo = changed_repo(:repository3, owner_uuid: users(:system_user).uuid,
                        name: "foo3")
    assert(repo.save)
  end

  test 'write permission allows changing modified_at' do
    act_as_user users(:active) do
      r = repositories(:foo)
      modtime_was = r.modified_at
      r.modified_at = Time.now
      assert r.save
      assert_operator modtime_was, :<, r.modified_at
    end
  end

  test 'write permission necessary for changing modified_at' do
    act_as_user users(:spectator) do
      r = repositories(:foo)
      modtime_was = r.modified_at
      r.modified_at = Time.now
      assert_raises ArvadosModel::PermissionDeniedError do
        r.save!
      end
      r.reload
      assert_equal modtime_was, r.modified_at
    end
  end

  ### Renaming

  test "non-admin can rename own repo" do
    act_as_user users(:active) do
      assert repositories(:foo).update_attributes(name: 'active/foo12345')
    end
  end

  test "top level repo can be touched by non-admin with can_manage" do
    add_permission_link users(:active), repositories(:arvados), 'can_manage'
    act_as_user users(:active) do
      assert changed_repo(:arvados, modified_at: Time.now).save
    end
  end

  test "top level repo cannot be renamed by non-admin with can_manage" do
    add_permission_link users(:active), repositories(:arvados), 'can_manage'
    act_as_user users(:active) do
      assert_not_allowed { changed_repo(:arvados, name: 'xarvados').save }
    end
  end
end
