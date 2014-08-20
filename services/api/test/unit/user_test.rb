require 'test_helper'

class UserTest < ActiveSupport::TestCase
  include CurrentApiClient

  # The fixture services/api/test/fixtures/users.yml serves as the input for this test case
  setup do
    # Make sure system_user exists before making "pre-test users" list
    system_user
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

    assert @uninvited_user.groups_i_can(:read).size == 1, "inactive and uninvited user can only read anonymous user group"
    assert @uninvited_user.groups_i_can(:read).first.ends_with? 'anonymouspublic' , "inactive and uninvited user can only read anonymous user group"
    assert @uninvited_user.groups_i_can(:write).size == 0, "inactive and uninvited user should not be able write to any groups"
    assert @uninvited_user.groups_i_can(:manage).size == 0, "inactive and uninvited user should not be able manage any groups"
  end

  test "find user method checks" do
    User.find(:all).each do |user|
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

    @all_users = User.find(:all)

    user = User.new
    user.first_name = "first_name_for_newly_created_user"
    user.save

    # verify there is one extra user in the db now
    assert_equal @all_users.size+1, User.find(:all).size

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

  test "create new user with notifications" do
    set_user_from_auth :admin

    create_user_and_verify_setup_and_notifications true, 'active-notify-address@example.com', 'inactive-notify-address@example.com', nil, false
    create_user_and_verify_setup_and_notifications true, 'active-notify-address@example.com', [], nil, false
    create_user_and_verify_setup_and_notifications true, [], [], nil, false
    create_user_and_verify_setup_and_notifications false, 'active-notify-address@example.com', 'inactive-notify-address@example.com', nil, false
    create_user_and_verify_setup_and_notifications false, [], 'inactive-notify-address@example.com', nil, false
    create_user_and_verify_setup_and_notifications false, [], [], nil, false
  end

  [
    [false, 'active-notify@example.com', 'inactive-notify@example.com', 'inactive-none@example.com', false, false, true],
    [false, 'active-notify@example.com', 'inactive-notify@example.com', 'inactive-vm@example.com', true, false, true],
    [false, 'active-notify@example.com', 'inactive-notify@example.com', 'inactive-repo@example.com', false, true, true],
    [false, 'active-notify@example.com', 'inactive-notify@example.com', 'inactive-both@example.com', true, true, true],

    [false, [], [], 'inactive-none-no-notifications@example.com', false, false, true],
    [false, [], [], 'inactive-vm-no-notifications@example.com', true, false, true],
    [false, [], [], 'inactive-repo-no-notifications@example.com', false, true, true],
    [false, [], [], 'inactive-both-no-notifications@example.com', true, true, true],

    [true, 'active-notify@example.com', 'inactive-notify@example.com', 'active-none@example.com', false, false, true],
    [true, 'active-notify@example.com', 'inactive-notify@example.com', 'active-vm@example.com', true, false, true],
    [true, 'active-notify@example.com', 'inactive-notify@example.com', 'active-repo@example.com', false, true, true],
    [true, 'active-notify@example.com', 'inactive-notify@example.com', 'active-both@example.com', true, true, true],

    [true, [], [], 'active-none-no-notifications@example.com', false, false, true],
    [true, [], [], 'active-vm-no-notifications@example.com', true, false, true],
    [true, [], [], 'active-notify-no-notifications@example.com', 'inactive-repo@example.com', false, true, true],
    [true, [], [], 'active-both-no-notifications@example.com', true, true, true],

    [false, [], [], nil, true, true, false],
    [false, [], [], 'arvados', true, true, false],
    [false, [], [], '@example.com', true, true, false],
    [false, [], [], '^^incorrect_format@example.com', true, true, false],

    [false, 'active-notify@example.com', 'inactive-notify@example.com', 'foo@example.com', true, true, true],  # existing repository name 'foo'
    [false, 'active-notify@example.com', 'inactive-notify@example.com', 'foo@example.com', true, false, true],  # existing repository name 'foo'
    [false, 'active-notify@example.com', 'inactive-notify@example.com', 'foo@example.com', false, true, true],  # existing repository name 'foo'
    [false, 'active-notify@example.com', 'inactive-notify@example.com', 'foo@example.com', false, false, true],  # existing repository name 'foo', but we are not creating repo or login link
    [false, 'active-notify@example.com', 'inactive-notify@example.com', 'xyz_can_login_to_vm@example.com', true, true, true], # existing vm login name
    [false, 'active-notify@example.com', 'inactive-notify@example.com', 'xyz_can_login_to_vm@example.com', true, false, true], # existing vm login name
    [false, 'active-notify@example.com', 'inactive-notify@example.com', 'xyz_can_login_to_vm@example.com', false, true, true], # existing vm login name
    [false, 'active-notify@example.com', 'inactive-notify@example.com', 'xyz_can_login_to_vm@example.com', false, false, true], # existing vm login name, but we are not creating repo or login link
  ].each do |active, active_recipients, inactive_recipients, email, auto_setup_vm, auto_setup_repo, valid_email_format|
    test "create new user with auto setup #{email} #{auto_setup_vm} #{auto_setup_repo}" do
      auto_setup_new_users = Rails.configuration.auto_setup_new_users
      auto_setup_new_users_with_vm_uuid = Rails.configuration.auto_setup_new_users_with_vm_uuid
      auto_setup_new_users_with_repository = Rails.configuration.auto_setup_new_users_with_repository

      begin
        set_user_from_auth :admin

        Rails.configuration.auto_setup_new_users = true

        if auto_setup_vm
          Rails.configuration.auto_setup_new_users_with_vm_uuid = virtual_machines(:testvm)['uuid']
        else
          Rails.configuration.auto_setup_new_users_with_vm_uuid = false
        end

        Rails.configuration.auto_setup_new_users_with_repository = auto_setup_repo

        create_user_and_verify_setup_and_notifications active, active_recipients, inactive_recipients, email, valid_email_format
      ensure
        Rails.configuration.auto_setup_new_users = auto_setup_new_users
        Rails.configuration.auto_setup_new_users_with_vm_uuid = auto_setup_new_users_with_vm_uuid
        Rails.configuration.auto_setup_new_users_with_repository = auto_setup_new_users_with_repository
      end
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
    User.find(:all).each do |user|
      if user.uuid == active_user_uuid
        found_deleted_user = true
        break
      end
    end
    assert !found_deleted_user, "found deleted user: "+active_user_uuid

  end

  test "create new user as non-admin user" do
    set_user_from_auth :active

    begin
      user = User.new
      user.save
    rescue ArvadosModel::PermissionDeniedError => e
    end
    assert (e.message.include? 'PermissionDeniedError'),
        'Expected PermissionDeniedError'
  end

  test "setup new user" do
    set_user_from_auth :admin

    email = 'foo@example.com'
    openid_prefix = 'http://openid/prefix'

    user = User.create ({uuid: 'zzzzz-tpzed-abcdefghijklmno', email: email})

    vm = VirtualMachine.create

    response = User.setup user, openid_prefix, 'test_repo', vm.uuid

    resp_user = find_obj_in_resp response, 'User'
    verify_user resp_user, email

    oid_login_perm = find_obj_in_resp response, 'Link', 'arvados#user'

    verify_link oid_login_perm, 'permission', 'can_login', resp_user[:email],
        resp_user[:uuid]

    assert_equal openid_prefix, oid_login_perm[:properties]['identity_url_prefix'],
        'expected identity_url_prefix not found for oid_login_perm'

    group_perm = find_obj_in_resp response, 'Link', 'arvados#group'
    verify_link group_perm, 'permission', 'can_read', resp_user[:uuid], nil

    repo_perm = find_obj_in_resp response, 'Link', 'arvados#repository'
    verify_link repo_perm, 'permission', 'can_manage', resp_user[:uuid], nil

    vm_perm = find_obj_in_resp response, 'Link', 'arvados#virtualMachine'
    verify_link vm_perm, 'permission', 'can_login', resp_user[:uuid], vm.uuid
  end

  test "setup new user with junk in database" do
    set_user_from_auth :admin

    email = 'foo@example.com'
    openid_prefix = 'http://openid/prefix'

    user = User.create ({uuid: 'zzzzz-tpzed-abcdefghijklmno', email: email})

    vm = VirtualMachine.create

    # Set up the bogus Link
    bad_uuid = 'zzzzz-tpzed-xyzxyzxyzxyzxyz'

    resp_link = Link.create ({tail_uuid: email, link_class: 'permission',
        name: 'can_login', head_uuid: bad_uuid})
    resp_link.save(validate: false)

    verify_link resp_link, 'permission', 'can_login', email, bad_uuid

    response = User.setup user, openid_prefix, 'test_repo', vm.uuid

    resp_user = find_obj_in_resp response, 'User'
    verify_user resp_user, email

    oid_login_perm = find_obj_in_resp response, 'Link', 'arvados#user'

    verify_link oid_login_perm, 'permission', 'can_login', resp_user[:email],
        resp_user[:uuid]

    assert_equal openid_prefix, oid_login_perm[:properties]['identity_url_prefix'],
        'expected identity_url_prefix not found for oid_login_perm'

    group_perm = find_obj_in_resp response, 'Link', 'arvados#group'
    verify_link group_perm, 'permission', 'can_read', resp_user[:uuid], nil

    repo_perm = find_obj_in_resp response, 'Link', 'arvados#repository'
    verify_link repo_perm, 'permission', 'can_manage', resp_user[:uuid], nil

    vm_perm = find_obj_in_resp response, 'Link', 'arvados#virtualMachine'
    verify_link vm_perm, 'permission', 'can_login', resp_user[:uuid], vm.uuid
  end



  test "setup new user in multiple steps" do
    set_user_from_auth :admin

    email = 'foo@example.com'
    openid_prefix = 'http://openid/prefix'

    user = User.create ({uuid: 'zzzzz-tpzed-abcdefghijklmno', email: email})

    response = User.setup user, openid_prefix

    resp_user = find_obj_in_resp response, 'User'
    verify_user resp_user, email

    oid_login_perm = find_obj_in_resp response, 'Link', 'arvados#user'
    verify_link oid_login_perm, 'permission', 'can_login', resp_user[:email],
        resp_user[:uuid]
    assert_equal openid_prefix, oid_login_perm[:properties]['identity_url_prefix'],
        'expected identity_url_prefix not found for oid_login_perm'

    group_perm = find_obj_in_resp response, 'Link', 'arvados#group'
    verify_link group_perm, 'permission', 'can_read', resp_user[:uuid], nil

    # invoke setup again with repo_name
    response = User.setup user, openid_prefix, 'test_repo'
    resp_user = find_obj_in_resp response, 'User', nil
    verify_user resp_user, email
    assert_equal user.uuid, resp_user[:uuid], 'expected uuid not found'

    group_perm = find_obj_in_resp response, 'Link', 'arvados#group'
    verify_link group_perm, 'permission', 'can_read', resp_user[:uuid], nil

    repo_perm = find_obj_in_resp response, 'Link', 'arvados#repository'
    verify_link repo_perm, 'permission', 'can_manage', resp_user[:uuid], nil

    # invoke setup again with a vm_uuid
    vm = VirtualMachine.create

    response = User.setup user, openid_prefix, 'test_repo', vm.uuid

    resp_user = find_obj_in_resp response, 'User', nil
    verify_user resp_user, email
    assert_equal user.uuid, resp_user[:uuid], 'expected uuid not found'

    group_perm = find_obj_in_resp response, 'Link', 'arvados#group'
    verify_link group_perm, 'permission', 'can_read', resp_user[:uuid], nil

    repo_perm = find_obj_in_resp response, 'Link', 'arvados#repository'
    verify_link repo_perm, 'permission', 'can_manage', resp_user[:uuid], nil

    vm_perm = find_obj_in_resp response, 'Link', 'arvados#virtualMachine'
    verify_link vm_perm, 'permission', 'can_login', resp_user[:uuid], vm.uuid
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
        if ArvadosModel::resource_class_for_uuid(x['head_uuid']).kind == head_kind
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

  def create_user_and_verify_setup_and_notifications (active, active_recipients, inactive_recipients, email, valid_email_format)
    Rails.configuration.new_user_notification_recipients = active_recipients
    Rails.configuration.new_inactive_user_notification_recipients = inactive_recipients

    assert_equal active_recipients, Rails.configuration.new_user_notification_recipients
    assert_equal inactive_recipients, Rails.configuration.new_inactive_user_notification_recipients

    ActionMailer::Base.deliveries = []

    user = User.new
    user.first_name = "first_name_for_newly_created_user"
    user.email = email
    user.is_active = active
    user.save

    # check user setup
    group = Group.where(name: 'All users').select do |g|
      g[:uuid].match /-f+$/
    end.first

    username = email.partition('@')[0] if email

    if !Rails.configuration.auto_setup_new_users || !valid_email_format
      # verify that the user is not added to "All groups" by auto_setup
      verify_link_exists false, group[:uuid], user.uuid, 'permission', 'can_read', nil, nil

      # check oid login link not created by auto_setup
      verify_link_exists false, user.uuid, user.email, 'permission', 'can_login', nil, nil
    else
      # verify that auto_setup took place
      # verify that the user is added to "All groups"
      verify_link_exists true, group[:uuid], user.uuid, 'permission', 'can_read', nil, nil

      # check oid login link
      verify_link_exists true, user.uuid, user.email, 'permission', 'can_login', nil, nil

      username = user.email.partition('@')[0]

      # check vm uuid
      vm_uuid = Rails.configuration.auto_setup_new_users_with_vm_uuid
      if vm_uuid
        verify_link_exists true, vm_uuid, user.uuid, 'permission', 'can_login', 'username', username
      else
        verify_link_exists false, vm_uuid, user.uuid, 'permission', 'can_login', 'username', username
      end

      # check repo
      if Rails.configuration.auto_setup_new_users_with_repository
        repos = Repository.where('name like ?', "%#{username}%")
        assert_not_nil repos, 'repository not found'
        assert_equal true, repos.any?, 'repository not found'
        repo_uuids = []
        repos.each do |repo|
          repo_uuids << repo[:uuid]
        end
        verify_link_exists true, repo_uuids, user.uuid, 'permission', 'can_manage', nil, nil
      end
    end

    # check email notifications
    new_user_email = nil
    new_inactive_user_email = nil

    ActionMailer::Base.deliveries.each do |d|
      if d.subject == "#{Rails.configuration.email_subject_prefix}New user notification" then
        new_user_email = d
      elsif d.subject == "#{Rails.configuration.email_subject_prefix}New inactive user notification" then
        new_inactive_user_email = d
      end
    end

    if not active
      if not inactive_recipients.empty? then
        assert_not_nil new_inactive_user_email, 'Expected new inactive user email after setup'
        assert_equal Rails.configuration.user_notifier_email_from, new_inactive_user_email.from[0]
        assert_equal inactive_recipients, new_inactive_user_email.to[0]
        assert_equal "#{Rails.configuration.email_subject_prefix}New inactive user notification", new_inactive_user_email.subject
      else
        assert_nil new_inactive_user_email, 'Did not expect new inactive user email after setup'
      end
    end

    if active
      assert_nil new_inactive_user_email, 'Expected no inactive user email after setting up active user'
      if not active_recipients.empty? then
        assert_not_nil new_user_email, 'Expected new user email after setup'
        assert_equal Rails.configuration.user_notifier_email_from, new_user_email.from[0]
        assert_equal active_recipients, new_user_email.to[0]
        assert_equal "#{Rails.configuration.email_subject_prefix}New user notification", new_user_email.subject
      else
        assert_nil new_user_email, 'Did not expect new user email after setup'
      end
    end
    ActionMailer::Base.deliveries = []

  end

  def verify_link_exists link_exists, head_uuid, tail_uuid, link_class, link_name, property_name, property_value
    all_links = Link.where(head_uuid: head_uuid,
                           tail_uuid: tail_uuid,
                           link_class: link_class,
                           name: link_name)
    assert_equal link_exists, all_links.any?, "Link #{'not' if link_exists} found #{property_value}"
    if link_exists && property_name && property_value
      all_links.each do |link|
        assert_equal true, all_links.first.properties[property_name].start_with?(property_value), 'Property not found in link'
      end
    end
  end

end
