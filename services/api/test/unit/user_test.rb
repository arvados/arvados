# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

require 'test_helper'

class UserTest < ActiveSupport::TestCase
  include CurrentApiClient

  # The fixture services/api/test/fixtures/users.yml serves as the input for this test case
  setup do
    # Make sure system_user exists before making "pre-test users" list
    system_user
  end

  %w(a aa a0 aA Aa AA A0).each do |username|
    test "#{username.inspect} is a valid username" do
      user = User.new(username: username)
      assert(user.valid?)
    end
  end

  test "username is not required" do
    user = User.new(username: nil)
    assert(user.valid?)
  end

  test "username beginning with numeral is invalid" do
    user = User.new(username: "0a")
    refute(user.valid?)
  end

  "\\.-_/!@#$%^&*()[]{}".each_char do |bad_char|
    test "username containing #{bad_char.inspect} is invalid" do
      user = User.new(username: "bad#{bad_char}username")
      refute(user.valid?)
    end
  end

  test "username must be unique" do
    user = User.new(username: users(:active).username)
    refute(user.valid?)
  end

  test "non-admin can't update username" do
    set_user_from_auth :rominiadmin
    user = User.find_by_uuid(users(:rominiadmin).uuid)
    user.username = "selfupdate"
    assert_not_allowed { user.save }
  end

  def check_admin_username_change(fixture_name)
    set_user_from_auth :admin_trustedclient
    user = User.find_by_uuid(users(fixture_name).uuid)
    user.username = "newnamefromtest"
    assert(user.save)
  end

  test "admin can set username" do
    check_admin_username_change(:active_no_prefs)
  end

  test "admin can update username" do
    check_admin_username_change(:active)
  end

  test "admin can update own username" do
    check_admin_username_change(:admin)
  end

  def check_new_username_setting(email_name, expect_name)
    set_user_from_auth :admin
    user = User.create!(email: "#{email_name}@example.org")
    assert_equal(expect_name, user.username)
  end

  test "new username set from e-mail" do
    check_new_username_setting("dakota", "dakota")
  end

  test "new username set from e-mail with leading digits" do
    check_new_username_setting("1dakota9", "dakota9")
  end

  test "new username set from e-mail with punctuation" do
    check_new_username_setting("dakota.9", "dakota9")
  end

  test "new username set from e-mail with leading digits and punctuation" do
    check_new_username_setting("1.dakota.z", "dakotaz")
  end

  test "new username set from e-mail with extra part" do
    check_new_username_setting("dakota+arvados", "dakota")
  end

  test "new username set with deduplication" do
    name = users(:active).username
    check_new_username_setting(name, "#{name}2")
    check_new_username_setting(name, "#{name}3")
    # Insert some out-of-order conflicts, to ensure our "sort by
    # username, stop when we see a hole" strategy doesn't depend on
    # insert order.
    check_new_username_setting("#{name}13", "#{name}13")
    check_new_username_setting("#{name}5", "#{name}5")
    check_new_username_setting(name, "#{name}4")
    6.upto(12).each do |n|
      check_new_username_setting(name, "#{name}#{n}")
    end
  end

  test "new username set avoiding blacklist" do
    Rails.configuration.Users.AutoSetupUsernameBlacklist = ConfigLoader.to_OrderedOptions({"root"=>{}})
    check_new_username_setting("root", "root2")
  end

  test "no username set when no base available" do
    check_new_username_setting("_", nil)
  end

  test "updating username updates repository names" do
    set_user_from_auth :admin
    user = users(:active)
    user.username = "newtestname"
    assert(user.save, "username update failed")
    {foo: "newtestname/foo", repository2: "newtestname/foo2"}.
        each_pair do |repo_sym, expect_name|
      assert_equal(expect_name, repositories(repo_sym).name)
    end
  end

  test "admin can clear username when user owns no repositories" do
    set_user_from_auth :admin
    user = users(:spectator)
    user.username = nil
    assert(user.save)
    assert_nil(user.username)
  end

  test "admin can't clear username when user owns repositories" do
    set_user_from_auth :admin
    user = users(:active)
    user.username = nil
    assert_not_allowed { user.save }
    refute_empty(user.errors[:username])
  end

  test "failed username update doesn't change repository names" do
    set_user_from_auth :admin
    user = users(:active)
    user.username = users(:fuse).username
    assert_not_allowed { user.save }
    assert_equal("active/foo", repositories(:foo).name)
  end

  [[false, 'foo@example.com', true, nil],
   [false, 'bar@example.com', nil, true],
   [true, 'foo@example.com', true, nil],
   [true, 'bar@example.com', true, true],
   [false, '', nil, nil],
   [true, '', true, nil]
  ].each do |auto_admin_first_user_config, auto_admin_user_config, foo_should_be_admin, bar_should_be_admin|
    # In each case, 'foo' is created first, then 'bar', then 'bar2', then 'baz'.
    test "auto admin with auto_admin_first=#{auto_admin_first_user_config} auto_admin=#{auto_admin_user_config}" do

      if auto_admin_first_user_config
        # This test requires no admin users exist (except for the system user)
        act_as_system_user do
          users(:admin).update_attributes!(is_admin: false)
        end
        @all_users = User.where("uuid not like '%-000000000000000'").where(:is_admin => true)
        assert_equal 0, @all_users.count, "No admin users should exist (except for the system user)"
      end

      Rails.configuration.Users.AutoAdminFirstUser = auto_admin_first_user_config
      Rails.configuration.Users.AutoAdminUserWithEmail = auto_admin_user_config

      # See if the foo user has is_admin
      foo = User.new
      foo.first_name = 'foo'
      foo.email = 'foo@example.com'

      act_as_system_user do
        foo.save!
      end

      foo = User.find(foo.id)   # get the user back
      assert_equal foo_should_be_admin, foo.is_admin, "is_admin is wrong for user foo"
      assert_equal 'foo', foo.first_name

      # See if the bar user has is_admin
      bar = User.new
      bar.first_name = 'bar'
      bar.email = 'bar@example.com'

      act_as_system_user do
        bar.save!
      end

      bar = User.find(bar.id)   # get the user back
      assert_equal bar_should_be_admin, bar.is_admin, "is_admin is wrong for user bar"
      assert_equal 'bar', bar.first_name

      # A subsequent user with the bar@example.com address should never be
      # elevated to admin
      bar2 = User.new
      bar2.first_name = 'bar2'
      bar2.email = 'bar@example.com'

      act_as_system_user do
        bar2.save!
      end

      bar2 = User.find(bar2.id)   # get the user back
      assert !bar2.is_admin, "is_admin is wrong for user bar2"
      assert_equal 'bar2', bar2.first_name

      # An ordinary new user should not be elevated to admin
      baz = User.new
      baz.first_name = 'baz'
      baz.email = 'baz@example.com'

      act_as_system_user do
        baz.save!
      end

      baz = User.find(baz.id)   # get the user back
      assert !baz.is_admin
      assert_equal 'baz', baz.first_name

    end
  end

  test "check non-admin active user properties" do
    @active_user = users(:active)     # get the active user
    assert !@active_user.is_admin, 'is_admin should not be set for a non-admin user'
    assert @active_user.is_active, 'user should be active'
    assert @active_user.is_invited, 'is_invited should be set'
    assert_not_nil @active_user.prefs, "user's preferences should be non-null, but may be size zero"
    assert (@active_user.can? :read=>"#{@active_user.uuid}"), "user should be able to read own object"
    assert (@active_user.can? :write=>"#{@active_user.uuid}"), "user should be able to write own object"
    assert (@active_user.can? :manage=>"#{@active_user.uuid}"), "user should be able to manage own object"

    assert @active_user.groups_i_can(:read).size > 0, "active user should be able read at least one group"

    # non-admin user cannot manage or write other user objects
    @uninvited_user = users(:inactive_uninvited)     # get the uninvited user
    assert !(@active_user.can? :read=>"#{@uninvited_user.uuid}")
    assert !(@active_user.can? :write=>"#{@uninvited_user.uuid}")
    assert !(@active_user.can? :manage=>"#{@uninvited_user.uuid}")
  end

  test "check admin user properties" do
    @admin_user = users(:admin)     # get the admin user
    assert @admin_user.is_admin, 'is_admin should be set for admin user'
    assert @admin_user.is_active, 'admin user cannot be inactive'
    assert @admin_user.is_invited, 'is_invited should be set'
    assert_not_nil @admin_user.uuid.size, "user's uuid should be non-null"
    assert_not_nil @admin_user.prefs, "user's preferences should be non-null, but may be size zero"
    assert @admin_user.identity_url.size > 0, "user's identity url is expected"
    assert @admin_user.can? :read=>"#{@admin_user.uuid}"
    assert @admin_user.can? :write=>"#{@admin_user.uuid}"
    assert @admin_user.can? :manage=>"#{@admin_user.uuid}"

    assert @admin_user.groups_i_can(:read).size > 0, "admin active user should be able read at least one group"
    assert @admin_user.groups_i_can(:write).size > 0, "admin active user should be able write to at least one group"
    assert @admin_user.groups_i_can(:manage).size > 0, "admin active user should be able manage at least one group"

    # admin user can also write or manage other users
    @uninvited_user = users(:inactive_uninvited)     # get the uninvited user
    assert @admin_user.can? :read=>"#{@uninvited_user.uuid}"
    assert @admin_user.can? :write=>"#{@uninvited_user.uuid}"
    assert @admin_user.can? :manage=>"#{@uninvited_user.uuid}"
  end

  test "check inactive and uninvited user properties" do
    @uninvited_user = users(:inactive_uninvited)     # get the uninvited user
    assert !@uninvited_user.is_admin, 'is_admin should not be set for a non-admin user'
    assert !@uninvited_user.is_active, 'user should be inactive'
    assert !@uninvited_user.is_invited, 'is_invited should not be set'
    assert @uninvited_user.can? :read=>"#{@uninvited_user.uuid}"
    assert @uninvited_user.can? :write=>"#{@uninvited_user.uuid}"
    assert @uninvited_user.can? :manage=>"#{@uninvited_user.uuid}"

    assert_equal(@uninvited_user.groups_i_can(:read).sort,
                 [@uninvited_user.uuid, groups(:anonymous_group).uuid].sort)
    assert_equal(@uninvited_user.groups_i_can(:write),
                 [@uninvited_user.uuid])
    assert_equal(@uninvited_user.groups_i_can(:manage),
                 [@uninvited_user.uuid])
  end

  test "find user method checks" do
    User.all.each do |user|
      assert_not_nil user.uuid, "non-null uuid expected for " + user.full_name
    end

    user = users(:active)     # get the active user

    found_user = User.find(user.id)   # find a user by the row id

    assert_equal found_user.full_name, user.first_name + ' ' + user.last_name
    assert_equal found_user.identity_url, user.identity_url
  end

  test "full name should not contain spurious whitespace" do
    set_user_from_auth :admin

    user = User.create ({uuid: 'zzzzz-tpzed-abcdefghijklmno', email: 'foo@example.com' })

    assert_equal '', user.full_name

    user.first_name = 'John'
    user.last_name = 'Smith'

    assert_equal user.first_name + ' ' + user.last_name, user.full_name
  end

  test "create new user" do
    set_user_from_auth :admin

    @all_users = User.all.to_a

    user = User.new
    user.first_name = "first_name_for_newly_created_user"
    user.save

    # verify there is one extra user in the db now
    assert_equal @all_users.size+1, User.all.count

    user = User.find(user.id)   # get the user back
    assert_equal(user.first_name, 'first_name_for_newly_created_user')
    assert_not_nil user.uuid, 'uuid should be set for newly created user'
    assert_nil user.email, 'email should be null for newly created user, because it was not passed in'
    assert_nil user.identity_url, 'identity_url should be null for newly created user, because it was not passed in'

    user.first_name = 'first_name_for_newly_created_user_updated'
    user.save
    user = User.find(user.id)   # get the user back
    assert_equal(user.first_name, 'first_name_for_newly_created_user_updated')
  end

  active_notify_list = ConfigLoader.to_OrderedOptions({"active-notify@example.com"=>{}})
  inactive_notify_list = ConfigLoader.to_OrderedOptions({"inactive-notify@example.com"=>{}})
  empty_notify_list = ConfigLoader.to_OrderedOptions({})

  test "create new user with notifications" do
    set_user_from_auth :admin

    create_user_and_verify_setup_and_notifications true, active_notify_list, inactive_notify_list, nil, nil
    create_user_and_verify_setup_and_notifications true, active_notify_list, empty_notify_list, nil, nil
    create_user_and_verify_setup_and_notifications true, empty_notify_list, empty_notify_list, nil, nil
    create_user_and_verify_setup_and_notifications false, active_notify_list, inactive_notify_list, nil, nil
    create_user_and_verify_setup_and_notifications false, empty_notify_list, inactive_notify_list, nil, nil
    create_user_and_verify_setup_and_notifications false, empty_notify_list, empty_notify_list, nil, nil
  end

  [
    # Easy inactive user tests.
    [false, empty_notify_list, empty_notify_list, "inactive-none@example.com", false, false, "inactivenone"],
    [false, empty_notify_list, empty_notify_list, "inactive-vm@example.com", true, false, "inactivevm"],
    [false, empty_notify_list, empty_notify_list, "inactive-repo@example.com", false, true, "inactiverepo"],
    [false, empty_notify_list, empty_notify_list, "inactive-both@example.com", true, true, "inactiveboth"],

    # Easy active user tests.
    [true, active_notify_list, inactive_notify_list, "active-none@example.com", false, false, "activenone"],
    [true, active_notify_list, inactive_notify_list, "active-vm@example.com", true, false, "activevm"],
    [true, active_notify_list, inactive_notify_list, "active-repo@example.com", false, true, "activerepo"],
    [true, active_notify_list, inactive_notify_list, "active-both@example.com", true, true, "activeboth"],

    # Test users with malformed e-mail addresses.
    [false, empty_notify_list, empty_notify_list, nil, true, true, nil],
    [false, empty_notify_list, empty_notify_list, "arvados", true, true, nil],
    [false, empty_notify_list, empty_notify_list, "@example.com", true, true, nil],
    [true, active_notify_list, inactive_notify_list, "*!*@example.com", true, false, nil],
    [true, active_notify_list, inactive_notify_list, "*!*@example.com", false, false, nil],

    # Test users with various username transformations.
    [false, empty_notify_list, empty_notify_list, "arvados@example.com", false, false, "arvados2"],
    [true, active_notify_list, inactive_notify_list, "arvados@example.com", false, false, "arvados2"],
    [true, active_notify_list, inactive_notify_list, "root@example.com", true, false, "root2"],
    [false, active_notify_list, inactive_notify_list, "root@example.com", true, false, "root2"],
    [true, active_notify_list, inactive_notify_list, "roo_t@example.com", false, true, "root2"],
    [false, empty_notify_list, empty_notify_list, "^^incorrect_format@example.com", true, true, "incorrectformat"],
    [true, active_notify_list, inactive_notify_list, "&4a_d9.@example.com", true, true, "ad9"],
    [true, active_notify_list, inactive_notify_list, "&4a_d9.@example.com", false, false, "ad9"],
    [false, active_notify_list, inactive_notify_list, "&4a_d9.@example.com", true, true, "ad9"],
    [false, active_notify_list, inactive_notify_list, "&4a_d9.@example.com", false, false, "ad9"],
  ].each do |active, new_user_recipients, inactive_recipients, email, auto_setup_vm, auto_setup_repo, expect_username|
    test "create new user with auto setup active=#{active} email=#{email} vm=#{auto_setup_vm} repo=#{auto_setup_repo}" do
      set_user_from_auth :admin

      Rails.configuration.Users.AutoSetupNewUsers = true

      if auto_setup_vm
        Rails.configuration.Users.AutoSetupNewUsersWithVmUUID = virtual_machines(:testvm)['uuid']
      else
        Rails.configuration.Users.AutoSetupNewUsersWithVmUUID = ""
      end

      Rails.configuration.Users.AutoSetupNewUsersWithRepository = auto_setup_repo

      create_user_and_verify_setup_and_notifications active, new_user_recipients, inactive_recipients, email, expect_username
    end
  end

  test "update existing user" do
    set_user_from_auth :active    # set active user as current user

    @active_user = users(:active)     # get the active user

    @active_user.first_name = "first_name_changed"
    @active_user.save

    @active_user = User.find(@active_user.id)   # get the user back
    assert_equal(@active_user.first_name, 'first_name_changed')

    # admin user also should be able to update the "active" user info
    set_user_from_auth :admin # set admin user as current user
    @active_user.first_name = "first_name_changed_by_admin_for_active_user"
    @active_user.save

    @active_user = User.find(@active_user.id)   # get the user back
    assert_equal(@active_user.first_name, 'first_name_changed_by_admin_for_active_user')
  end

  test "delete a user and verify" do
    @active_user = users(:active)     # get the active user
    active_user_uuid = @active_user.uuid

    set_user_from_auth :admin
    @active_user.delete

    found_deleted_user = false
    User.all.each do |user|
      if user.uuid == active_user_uuid
        found_deleted_user = true
        break
      end
    end
    assert !found_deleted_user, "found deleted user: "+active_user_uuid

  end

  test "create new user as non-admin user" do
    set_user_from_auth :active
    assert_not_allowed { User.new.save }
  end

  [true, false].each do |visible|
    test "setup new user with ActivatedUsersAreVisibleToOthers=#{visible}" do
      Rails.configuration.Users.ActivatedUsersAreVisibleToOthers = visible
      set_user_from_auth :admin

      email = 'foo@example.com'

      user = User.create ({uuid: 'zzzzz-tpzed-abcdefghijklmno', email: email})

      vm = VirtualMachine.create

      response = user.setup(repo_name: 'foo/testrepo',
                            vm_uuid: vm.uuid)

      resp_user = find_obj_in_resp response, 'User'
      verify_user resp_user, email

      group_perm = find_obj_in_resp response, 'Link', 'arvados#group'
      verify_link group_perm, 'permission', 'can_read', resp_user[:uuid], nil

      group_perm2 = find_obj_in_resp response, 'Link', 'arvados#user'
      if visible
        verify_link group_perm2, 'permission', 'can_read', groups(:all_users).uuid, nil
      else
        assert_nil group_perm2
      end

      repo_perm = find_obj_in_resp response, 'Link', 'arvados#repository'
      verify_link repo_perm, 'permission', 'can_manage', resp_user[:uuid], nil

      vm_perm = find_obj_in_resp response, 'Link', 'arvados#virtualMachine'
      verify_link vm_perm, 'permission', 'can_login', resp_user[:uuid], vm.uuid
      assert_equal("foo", vm_perm.properties["username"])
    end
  end

  test "setup new user with junk in database" do
    set_user_from_auth :admin

    email = 'foo@example.com'

    user = User.create ({uuid: 'zzzzz-tpzed-abcdefghijklmno', email: email})

    vm = VirtualMachine.create

    response = user.setup(repo_name: 'foo/testrepo',
                          vm_uuid: vm.uuid)

    resp_user = find_obj_in_resp response, 'User'
    verify_user resp_user, email

    group_perm = find_obj_in_resp response, 'Link', 'arvados#group'
    verify_link group_perm, 'permission', 'can_read', resp_user[:uuid], nil

    repo_perm = find_obj_in_resp response, 'Link', 'arvados#repository'
    verify_link repo_perm, 'permission', 'can_manage', resp_user[:uuid], nil

    vm_perm = find_obj_in_resp response, 'Link', 'arvados#virtualMachine'
    verify_link vm_perm, 'permission', 'can_login', resp_user[:uuid], vm.uuid
    assert_equal("foo", vm_perm.properties["username"])
  end

  test "setup new user in multiple steps" do
    set_user_from_auth :admin

    email = 'foo@example.com'

    user = User.create ({uuid: 'zzzzz-tpzed-abcdefghijklmno', email: email})

    response = user.setup()

    resp_user = find_obj_in_resp response, 'User'
    verify_user resp_user, email

    group_perm = find_obj_in_resp response, 'Link', 'arvados#group'
    verify_link group_perm, 'permission', 'can_read', resp_user[:uuid], nil

    group_perm2 = find_obj_in_resp response, 'Link', 'arvados#user'
    verify_link group_perm2, 'permission', 'can_read', groups(:all_users).uuid, nil

    # invoke setup again with repo_name
    response = user.setup(repo_name: 'foo/testrepo')
    resp_user = find_obj_in_resp response, 'User', nil
    verify_user resp_user, email
    assert_equal user.uuid, resp_user[:uuid], 'expected uuid not found'

    group_perm = find_obj_in_resp response, 'Link', 'arvados#group'
    verify_link group_perm, 'permission', 'can_read', resp_user[:uuid], nil

    repo_perm = find_obj_in_resp response, 'Link', 'arvados#repository'
    verify_link repo_perm, 'permission', 'can_manage', resp_user[:uuid], nil

    # invoke setup again with a vm_uuid
    vm = VirtualMachine.create

    response = user.setup(repo_name: 'foo/testrepo',
                          vm_uuid: vm.uuid)

    resp_user = find_obj_in_resp response, 'User', nil
    verify_user resp_user, email
    assert_equal user.uuid, resp_user[:uuid], 'expected uuid not found'

    group_perm = find_obj_in_resp response, 'Link', 'arvados#group'
    verify_link group_perm, 'permission', 'can_read', resp_user[:uuid], nil

    repo_perm = find_obj_in_resp response, 'Link', 'arvados#repository'
    verify_link repo_perm, 'permission', 'can_manage', resp_user[:uuid], nil

    vm_perm = find_obj_in_resp response, 'Link', 'arvados#virtualMachine'
    verify_link vm_perm, 'permission', 'can_login', resp_user[:uuid], vm.uuid
    assert_equal("foo", vm_perm.properties["username"])
  end

  def find_obj_in_resp (response_items, object_type, head_kind=nil)
    return_obj = nil
    response_items.each { |x|
      if !x
        next
      end

      if object_type == 'User'
        if ArvadosModel::resource_class_for_uuid(x['uuid']) == User
          return_obj = x
          break
        end
      else  # looking for a link
        if ArvadosModel::resource_class_for_uuid(x['head_uuid']).andand.kind == head_kind
          return_obj = x
          break
        end
      end
    }
    return return_obj
  end

  def verify_user (resp_user, email)
    assert_not_nil resp_user, 'expected user object'
    assert_not_nil resp_user['uuid'], 'expected user object'
    assert_equal email, resp_user['email'], 'expected email not found'
  end

  def verify_link (link_object, link_class, link_name, tail_uuid, head_uuid)
    assert_not_nil link_object, "expected link for #{link_class} #{link_name}"
    assert_not_nil link_object[:uuid],
        "expected non-nil uuid for link for #{link_class} #{link_name}"
    assert_equal link_class, link_object[:link_class],
        "expected link_class not found for #{link_class} #{link_name}"
    assert_equal link_name, link_object[:name],
        "expected link_name not found for #{link_class} #{link_name}"
    assert_equal tail_uuid, link_object[:tail_uuid],
        "expected tail_uuid not found for #{link_class} #{link_name}"
    if head_uuid
      assert_equal head_uuid, link_object[:head_uuid],
          "expected head_uuid not found for #{link_class} #{link_name}"
    end
  end

  def create_user_and_verify_setup_and_notifications (active, new_user_recipients, inactive_recipients, email, expect_username)
    Rails.configuration.Users.NewUserNotificationRecipients = new_user_recipients
    Rails.configuration.Users.NewInactiveUserNotificationRecipients = inactive_recipients

    ActionMailer::Base.deliveries = []

    can_setup = (Rails.configuration.Users.AutoSetupNewUsers and
                 (not expect_username.nil?))
    expect_repo_name = "#{expect_username}/#{expect_username}"
    prior_repo = Repository.where(name: expect_repo_name).first

    user = User.new
    user.first_name = "first_name_for_newly_created_user"
    user.email = email
    user.is_active = active
    user.save!
    assert_equal(expect_username, user.username)

    # check user setup
    verify_link_exists(Rails.configuration.Users.AutoSetupNewUsers || active,
                       groups(:all_users).uuid, user.uuid,
                       "permission", "can_read")

    # Check for repository.
    if named_repo = (prior_repo or
                     Repository.where(name: expect_repo_name).first)
      verify_link_exists((can_setup and prior_repo.nil? and
                          Rails.configuration.Users.AutoSetupNewUsersWithRepository),
                         named_repo.uuid, user.uuid, "permission", "can_manage")
    end

    # Check for VM login.
    if (auto_vm_uuid = Rails.configuration.Users.AutoSetupNewUsersWithVmUUID) != ""
      verify_link_exists(can_setup, auto_vm_uuid, user.uuid,
                         "permission", "can_login", "username", expect_username)
    end

    # check email notifications
    new_user_email = nil
    new_inactive_user_email = nil

    new_user_email_subject = "#{Rails.configuration.Users.EmailSubjectPrefix}New user created notification"
    if Rails.configuration.Users.AutoSetupNewUsers
      new_user_email_subject = (expect_username or active) ?
                                 "#{Rails.configuration.Users.EmailSubjectPrefix}New user created and setup notification" :
                                 "#{Rails.configuration.Users.EmailSubjectPrefix}New user created, but not setup notification"
    end

    ActionMailer::Base.deliveries.each do |d|
      if d.subject == new_user_email_subject then
        new_user_email = d
      elsif d.subject == "#{Rails.configuration.Users.EmailSubjectPrefix}New inactive user notification" then
        new_inactive_user_email = d
      end
    end

    # both active and inactive user creations should result in new user creation notification mails,
    # if the new user email recipients config parameter is set
    if not new_user_recipients.empty? then
      assert_not_nil new_user_email, 'Expected new user email after setup'
      assert_equal Rails.configuration.Users.UserNotifierEmailFrom, new_user_email.from[0]
      assert_equal new_user_recipients.stringify_keys.keys.first, new_user_email.to[0]
      assert_equal new_user_email_subject, new_user_email.subject
    else
      assert_nil new_user_email, 'Did not expect new user email after setup'
    end

    if not active
      if not inactive_recipients.empty? then
        assert_not_nil new_inactive_user_email, 'Expected new inactive user email after setup'
        assert_equal Rails.configuration.Users.UserNotifierEmailFrom, new_inactive_user_email.from[0]
        assert_equal inactive_recipients.stringify_keys.keys.first, new_inactive_user_email.to[0]
        assert_equal "#{Rails.configuration.Users.EmailSubjectPrefix}New inactive user notification", new_inactive_user_email.subject
      else
        assert_nil new_inactive_user_email, 'Did not expect new inactive user email after setup'
      end
    else
      assert_nil new_inactive_user_email, 'Expected no inactive user email after setting up active user'
    end
    ActionMailer::Base.deliveries = []
  end

  def verify_link_exists link_exists, head_uuid, tail_uuid, link_class, link_name, property_name=nil, property_value=nil
    all_links = Link.where(head_uuid: head_uuid,
                           tail_uuid: tail_uuid,
                           link_class: link_class,
                           name: link_name)
    assert_equal link_exists, all_links.any?, "Link#{' not' if link_exists} found for #{link_name} #{link_class} #{property_value}"
    if link_exists && property_name && property_value
      all_links.each do |link|
        assert_equal true, all_links.first.properties[property_name].start_with?(property_value), 'Property not found in link'
      end
    end
  end

  def assert_update_success(old_uuid:, new_uuid:, expect_owned_objects: true)
    [[User, :uuid],
     [Link, :head_uuid],
     [Link, :tail_uuid],
     [Group, :owner_uuid],
     [Collection, :owner_uuid],
    ].each do |klass, attr|
      assert_empty klass.where(attr => old_uuid)
      if klass == User || expect_owned_objects
        assert_not_empty klass.where(attr => new_uuid)
      end
    end
  end

  test "lookup user by email" do
    u = User.register({"email" => "active-user@arvados.local", "identity_url" => "different-identity-url"})
    active = User.find_by_uuid(users(:active).uuid)
    assert_equal active.uuid, u.uuid
    assert_equal "active-user@arvados.local", active.email
    # identity_url is not updated
    assert_equal "https://active-user.openid.local", active.identity_url
  end

  test "lookup user by alternate email" do
    # register method will find that active-user@arvados.local already
    # exists and return existing 'active' user.
    u = User.register({"email" => "user@parent-company.com",
                       "alternate_emails" => ["active-user@arvados.local"],
                       "identity_url" => "different-identity-url"})
    active = User.find_by_uuid(users(:active).uuid)
    assert_equal active.uuid, u.uuid

    # email should be updated
    assert_equal "user@parent-company.com", active.email

    # identity_url is not updated
    assert_equal "https://active-user.openid.local", active.identity_url
  end

  test "register new user" do
    u = User.register({"email" => "never-before-seen-user@arvados.local",
                       "identity_url" => "different-identity-url",
                       "first_name" => "Robert",
                       "last_name" => "Baratheon",
                       "username" => "bobby"})
    nbs = User.find_by_uuid(u.uuid)
    assert_equal nbs.uuid, u.uuid
    assert_equal "different-identity-url", nbs.identity_url
    assert_equal "never-before-seen-user@arvados.local", nbs.email
    assert_equal false, nbs.is_admin
    assert_equal false , nbs.is_active
    assert_equal "bobby", nbs.username
    assert_equal "Robert", nbs.first_name
    assert_equal "Baratheon", nbs.last_name
  end

  test "fail when email address is ambiguous" do
    User.register({"email" => "active-user@arvados.local"})
    u = User.register({"email" => "never-before-seen-user@arvados.local"})
    u.email = "active-user@arvados.local"
    act_as_system_user do
      u.save!
    end
    assert_raises do
      User.register({"email" => "active-user@arvados.local"})
    end
  end

  test "fail lookup without identifiers" do
    assert_raises do
      User.register({"first_name" => "Robert", "last_name" => "Baratheon"})
    end
    assert_raises do
      User.register({"first_name" => "Robert", "last_name" => "Baratheon", "identity_url" => "", "email" => ""})
    end
  end

  test "user can update name" do
    set_user_from_auth :active
    user = users(:active)
    user.first_name = "MyNewName"
    assert user.save
  end

  test "user cannot update email" do
    set_user_from_auth :active
    user = users(:active)
    user.email = "new-name@example.com"
    assert_not_allowed { user.save }
  end

  test "admin can update email" do
    set_user_from_auth :admin
    user = users(:active)
    user.email = "new-name@example.com"
    assert user.save
  end

end
