# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

require 'can_be_an_owner'

class User < ArvadosModel
  include HasUuid
  include KindAndEtag
  include CommonApiTemplate
  include CanBeAnOwner
  extend CurrentApiClient

  serialize :prefs, Hash
  has_many :api_client_authorizations
  validates(:username,
            format: {
              with: /\A[A-Za-z][A-Za-z0-9]*\z/,
              message: "must begin with a letter and contain only alphanumerics",
            },
            uniqueness: true,
            allow_nil: true)
  validate :must_unsetup_to_deactivate
  before_update :prevent_privilege_escalation
  before_update :prevent_inactive_admin
  before_update :verify_repositories_empty, :if => Proc.new {
    username.nil? and username_changed?
  }
  after_update :setup_on_activate

  before_create :check_auto_admin
  before_create :set_initial_username, :if => Proc.new {
    username.nil? and email
  }
  after_create :after_ownership_change
  after_create :setup_on_activate
  after_create :add_system_group_permission_link
  after_create :auto_setup_new_user, :if => Proc.new {
    Rails.configuration.Users.AutoSetupNewUsers and
    (uuid != system_user_uuid) and
    (uuid != anonymous_user_uuid) and
    (uuid[0..4] == Rails.configuration.ClusterID)
  }
  after_create :send_admin_notifications

  before_update :before_ownership_change
  after_update :after_ownership_change
  after_update :send_profile_created_notification
  after_update :sync_repository_names, :if => Proc.new {
    (uuid != system_user_uuid) and
    saved_change_to_username? and
    (not username_before_last_save.nil?)
  }
  before_destroy :clear_permissions
  after_destroy :remove_self_from_permissions

  has_many :authorized_keys, :foreign_key => :authorized_user_uuid, :primary_key => :uuid
  has_many :repositories, foreign_key: :owner_uuid, primary_key: :uuid

  default_scope { where('redirect_to_user_uuid is null') }

  api_accessible :user, extend: :common do |t|
    t.add :email
    t.add :username
    t.add :full_name
    t.add :first_name
    t.add :last_name
    t.add :identity_url
    t.add :is_active
    t.add :is_admin
    t.add :is_invited
    t.add :prefs
    t.add :writable_by
  end

  ALL_PERMISSIONS = {read: true, write: true, manage: true}

  # Map numeric permission levels (see lib/create_permission_view.sql)
  # back to read/write/manage flags.
  PERMS_FOR_VAL =
    [{},
     {read: true},
     {read: true, write: true},
     {read: true, write: true, manage: true}]

  VAL_FOR_PERM =
    {:read => 1,
     :write => 2,
     :manage => 3}


  def full_name
    "#{first_name} #{last_name}".strip
  end

  def is_invited
    !!(self.is_active ||
       Rails.configuration.Users.NewUsersAreActive ||
       self.groups_i_can(:read).select { |x| x.match(/-f+$/) }.first)
  end

  def groups_i_can(verb)
    my_groups = self.group_permissions(VAL_FOR_PERM[verb]).keys
    if verb == :read
      my_groups << anonymous_group_uuid
    end
    my_groups
  end

  def can?(actions)
    return true if is_admin
    actions.each do |action, target|
      unless target.nil?
        if target.respond_to? :uuid
          target_uuid = target.uuid
        else
          target_uuid = target
          target = ArvadosModel.find_by_uuid(target_uuid)
        end
      end
      next if target_uuid == self.uuid

      target_owner_uuid = target.owner_uuid if target.respond_to? :owner_uuid

      user_uuids_subquery = USER_UUIDS_SUBQUERY_TEMPLATE % {user: "$1", perm_level: "$3"}

      unless ActiveRecord::Base.connection.
        exec_query(%{
SELECT 1 FROM #{PERMISSION_VIEW}
  WHERE user_uuid in (#{user_uuids_subquery}) and
        ((target_uuid = $2 and perm_level >= $3)
         or (target_uuid = $4 and perm_level >= $3 and traverse_owned))
},
                  # "name" arg is a query label that appears in logs:
                   "user_can_query",
                   [[nil, self.uuid],
                    [nil, target_uuid],
                    [nil, VAL_FOR_PERM[action]],
                    [nil, target_owner_uuid]]
                  ).any?
        return false
      end
    end
    true
  end

  def before_ownership_change
    if owner_uuid_changed? and !self.owner_uuid_was.nil?
      MaterializedPermission.where(user_uuid: owner_uuid_was, target_uuid: uuid).delete_all
      update_permissions self.owner_uuid_was, self.uuid, REVOKE_PERM
    end
  end

  def after_ownership_change
    if saved_change_to_owner_uuid?
      update_permissions self.owner_uuid, self.uuid, CAN_MANAGE_PERM
    end
  end

  def clear_permissions
    MaterializedPermission.where("user_uuid = ? and target_uuid != ?", uuid, uuid).delete_all
  end

  def forget_cached_group_perms
    @group_perms = nil
  end

  def remove_self_from_permissions
    MaterializedPermission.where("target_uuid = ?", uuid).delete_all
    check_permissions_against_full_refresh
  end

  # Return a hash of {user_uuid: group_perms}
  #
  # note: this does not account for permissions that a user gains by
  # having can_manage on another user.
  def self.all_group_permissions
    all_perms = {}
    ActiveRecord::Base.connection.
      exec_query(%{
SELECT user_uuid, target_uuid, perm_level
                  FROM #{PERMISSION_VIEW}
                  WHERE traverse_owned
},
                  # "name" arg is a query label that appears in logs:
                 "all_group_permissions").
      rows.each do |user_uuid, group_uuid, max_p_val|
      all_perms[user_uuid] ||= {}
      all_perms[user_uuid][group_uuid] = PERMS_FOR_VAL[max_p_val.to_i]
    end
    all_perms
  end

  # Return a hash of {group_uuid: perm_hash} where perm_hash[:read]
  # and perm_hash[:write] are true if this user can read and write
  # objects owned by group_uuid.
  def group_permissions(level=1)
    @group_perms ||= {}
    if @group_perms.empty?
      user_uuids_subquery = USER_UUIDS_SUBQUERY_TEMPLATE % {user: "$1", perm_level: 1}

      ActiveRecord::Base.connection.
        exec_query(%{
SELECT target_uuid, perm_level
  FROM #{PERMISSION_VIEW}
  WHERE user_uuid in (#{user_uuids_subquery}) and perm_level >= 1
},
                   # "name" arg is a query label that appears in logs:
                   "User.group_permissions",
                   # "binds" arg is an array of [col_id, value] for '$1' vars:
                   [[nil, uuid]]).
        rows.each do |group_uuid, max_p_val|
        @group_perms[group_uuid] = PERMS_FOR_VAL[max_p_val.to_i]
      end
    end

    case level
    when 1
      @group_perms
    when 2
      @group_perms.select {|k,v| v[:write] }
    when 3
      @group_perms.select {|k,v| v[:manage] }
    else
      raise "level must be 1, 2 or 3"
    end
  end

  # create links
  def setup(repo_name: nil, vm_uuid: nil, send_notification_email: nil)
    newly_invited = Link.where(tail_uuid: self.uuid,
                              head_uuid: all_users_group_uuid,
                              link_class: 'permission',
                              name: 'can_read').empty?

    # Add can_read link from this user to "all users" which makes this
    # user "invited"
    group_perm = create_user_group_link

    # Add git repo
    repo_perm = if (!repo_name.nil? || Rails.configuration.Users.AutoSetupNewUsersWithRepository) and !username.nil?
                  repo_name ||= "#{username}/#{username}"
                  create_user_repo_link repo_name
                end

    # Add virtual machine
    if vm_uuid.nil? and !Rails.configuration.Users.AutoSetupNewUsersWithVmUUID.empty?
      vm_uuid = Rails.configuration.Users.AutoSetupNewUsersWithVmUUID
    end

    vm_login_perm = if vm_uuid && username
                      create_vm_login_permission_link(vm_uuid, username)
                    end

    # Send welcome email
    if send_notification_email.nil?
      send_notification_email = Rails.configuration.Mail.SendUserSetupNotificationEmail
    end

    if newly_invited and send_notification_email and !Rails.configuration.Users.UserSetupMailText.empty?
      begin
        UserNotifier.account_is_setup(self).deliver_now
      rescue => e
        logger.warn "Failed to send email to #{self.email}: #{e}"
      end
    end

    forget_cached_group_perms

    return [repo_perm, vm_login_perm, group_perm, self].compact
  end

  # delete user signatures, login, repo, and vm perms, and mark as inactive
  def unsetup
    # delete oid_login_perms for this user
    #
    # note: these permission links are obsolete, they have no effect
    # on anything and they are not created for new users.
    Link.where(tail_uuid: self.email,
                     link_class: 'permission',
                     name: 'can_login').destroy_all

    # delete repo_perms for this user
    Link.where(tail_uuid: self.uuid,
                     link_class: 'permission',
                     name: 'can_manage').destroy_all

    # delete vm_login_perms for this user
    Link.where(tail_uuid: self.uuid,
                     link_class: 'permission',
                     name: 'can_login').destroy_all

    # delete "All users" group read permissions for this user
    Link.where(tail_uuid: self.uuid,
                     head_uuid: all_users_group_uuid,
                     link_class: 'permission',
                     name: 'can_read').destroy_all

    # delete any signatures by this user
    Link.where(link_class: 'signature',
                     tail_uuid: self.uuid).destroy_all

    # delete tokens for this user
    ApiClientAuthorization.where(user_id: self.id).destroy_all
    # delete ssh keys for this user
    AuthorizedKey.where(owner_uuid: self.uuid).destroy_all
    AuthorizedKey.where(authorized_user_uuid: self.uuid).destroy_all

    # delete user preferences (including profile)
    self.prefs = {}

    # mark the user as inactive
    self.is_admin = false  # can't be admin and inactive
    self.is_active = false
    forget_cached_group_perms
    self.save!
  end

  def must_unsetup_to_deactivate
    if !self.new_record? &&
       self.uuid[0..4] == Rails.configuration.Login.LoginCluster &&
       self.uuid[0..4] != Rails.configuration.ClusterID
      # OK to update our local record to whatever the LoginCluster
      # reports, because self-activate is not allowed.
      return
    elsif self.is_active_changed? &&
       self.is_active_was &&
       !self.is_active

      # When a user is set up, they are added to the "All users"
      # group.  A user that is part of the "All users" group is
      # allowed to self-activate.
      #
      # It doesn't make sense to deactivate a user (set is_active =
      # false) without first removing them from the "All users" group,
      # because they would be able to immediately reactivate
      # themselves.
      #
      # The 'unsetup' method removes the user from the "All users"
      # group (and also sets is_active = false) so send a message
      # explaining the correct way to deactivate a user.
      #
      if Link.where(tail_uuid: self.uuid,
                    head_uuid: all_users_group_uuid,
                    link_class: 'permission',
                    name: 'can_read').any?
        errors.add :is_active, "cannot be set to false directly, use the 'Deactivate' button on Workbench, or the 'unsetup' API call"
      end
    end
  end

  def set_initial_username(requested: false)
    if !requested.is_a?(String) || requested.empty?
      email_parts = email.partition("@")
      local_parts = email_parts.first.partition("+")
      if email_parts.any?(&:empty?)
        return
      elsif not local_parts.first.empty?
        requested = local_parts.first
      else
        requested = email_parts.first
      end
    end
    requested.sub!(/^[^A-Za-z]+/, "")
    requested.gsub!(/[^A-Za-z0-9]/, "")
    unless requested.empty?
      self.username = find_usable_username_from(requested)
    end
  end

  # Move this user's (i.e., self's) owned items to new_owner_uuid and
  # new_user_uuid (for things normally owned directly by the user).
  #
  # If redirect_auth is true, also reassign auth tokens and ssh keys,
  # and redirect this account to redirect_to_user_uuid, i.e., when a
  # caller authenticates to this account in the future, the account
  # redirect_to_user_uuid account will be used instead.
  #
  # current_user must have admin privileges, i.e., the caller is
  # responsible for checking permission to do this.
  def merge(new_owner_uuid:, new_user_uuid:, redirect_to_new_user:)
    raise PermissionDeniedError if !current_user.andand.is_admin
    raise "Missing new_owner_uuid" if !new_owner_uuid
    raise "Missing new_user_uuid" if !new_user_uuid
    transaction(requires_new: true) do
      reload
      raise "cannot merge an already merged user" if self.redirect_to_user_uuid

      new_user = User.where(uuid: new_user_uuid).first
      raise "user does not exist" if !new_user
      raise "cannot merge to an already merged user" if new_user.redirect_to_user_uuid

      self.clear_permissions
      new_user.clear_permissions

      # If 'self' is a remote user, don't transfer authorizations
      # (i.e. ability to access the account) to the new user, because
      # that gives the remote site the ability to access the 'new'
      # user account that takes over the 'self' account.
      #
      # If 'self' is a local user, it is okay to transfer
      # authorizations, even if the 'new' user is a remote account,
      # because the remote site does not gain the ability to access an
      # account it could not before.

      if redirect_to_new_user and self.uuid[0..4] == Rails.configuration.ClusterID
        # Existing API tokens and ssh keys are updated to authenticate
        # to the new user.
        ApiClientAuthorization.
          where(user_id: id).
          update_all(user_id: new_user.id)

        user_updates = [
          [AuthorizedKey, :owner_uuid],
          [AuthorizedKey, :authorized_user_uuid],
          [Link, :owner_uuid],
          [Link, :tail_uuid],
          [Link, :head_uuid],
        ]
      else
        # Destroy API tokens and ssh keys associated with the old
        # user.
        ApiClientAuthorization.where(user_id: id).destroy_all
        AuthorizedKey.where(owner_uuid: uuid).destroy_all
        AuthorizedKey.where(authorized_user_uuid: uuid).destroy_all
        user_updates = [
          [Link, :owner_uuid],
          [Link, :tail_uuid]
        ]
      end

      # References to the old user UUID in the context of a user ID
      # (rather than a "home project" in the project hierarchy) are
      # updated to point to the new user.
      user_updates.each do |klass, column|
        klass.where(column => uuid).update_all(column => new_user.uuid)
      end

      # Need to update repository names to new username
      if username
        old_repo_name_re = /^#{Regexp.escape(username)}\//
        Repository.where(:owner_uuid => uuid).each do |repo|
          repo.owner_uuid = new_user.uuid
          repo_name_sub = "#{new_user.username}/"
          name = repo.name.sub(old_repo_name_re, repo_name_sub)
          while (conflict = Repository.where(:name => name).first) != nil
            repo_name_sub += "migrated"
            name = repo.name.sub(old_repo_name_re, repo_name_sub)
          end
          repo.name = name
          repo.save!
        end
      end

      # References to the merged user's "home project" are updated to
      # point to new_owner_uuid.
      ActiveRecord::Base.descendants.reject(&:abstract_class?).each do |klass|
        next if [ApiClientAuthorization,
                 AuthorizedKey,
                 Link,
                 Log,
                 Repository].include?(klass)
        next if !klass.columns.collect(&:name).include?('owner_uuid')
        klass.where(owner_uuid: uuid).update_all(owner_uuid: new_owner_uuid)
      end

      if redirect_to_new_user
        update_attributes!(redirect_to_user_uuid: new_user.uuid, username: nil)
      end
      skip_check_permissions_against_full_refresh do
        update_permissions self.uuid, self.uuid, CAN_MANAGE_PERM
        update_permissions new_user.uuid, new_user.uuid, CAN_MANAGE_PERM
        update_permissions new_user.owner_uuid, new_user.uuid, CAN_MANAGE_PERM
      end
      update_permissions self.owner_uuid, self.uuid, CAN_MANAGE_PERM
    end
  end

  def redirects_to
    user = self
    redirects = 0
    while (uuid = user.redirect_to_user_uuid)
      break if uuid.empty?
      nextuser = User.unscoped.find_by_uuid(uuid)
      if !nextuser
        raise Exception.new("user uuid #{user.uuid} redirects to nonexistent uuid '#{uuid}'")
      end
      user = nextuser
      redirects += 1
      if redirects > 15
        raise "Starting from #{self.uuid} redirect_to_user_uuid exceeded maximum number of redirects"
      end
    end
    user
  end

  def self.register info
    # login info expected fields, all can be optional but at minimum
    # must supply either 'identity_url' or 'email'
    #
    #   email
    #   first_name
    #   last_name
    #   username
    #   alternate_emails
    #   identity_url

    primary_user = nil

    # local database
    identity_url = info['identity_url']

    if identity_url && identity_url.length > 0
      # Only local users can create sessions, hence uuid_like_pattern
      # here.
      user = User.unscoped.where('identity_url = ? and uuid like ?',
                                 identity_url,
                                 User.uuid_like_pattern).first
      primary_user = user.redirects_to if user
    end

    if !primary_user
      # identity url is unset or didn't find matching record.
      emails = [info['email']] + (info['alternate_emails'] || [])
      emails.select! {|em| !em.nil? && !em.empty?}

      User.unscoped.where('email in (?) and uuid like ?',
                          emails,
                          User.uuid_like_pattern).each do |user|
        if !primary_user
          primary_user = user.redirects_to
        elsif primary_user.uuid != user.redirects_to.uuid
          raise "Ambiguous email address, directs to both #{primary_user.uuid} and #{user.redirects_to.uuid}"
        end
      end
    end

    if !primary_user
      # New user registration
      primary_user = User.new(:owner_uuid => system_user_uuid,
                              :is_admin => false,
                              :is_active => Rails.configuration.Users.NewUsersAreActive)

      primary_user.set_initial_username(requested: info['username']) if info['username'] && !info['username'].blank?
      primary_user.identity_url = info['identity_url'] if identity_url
    end

    primary_user.email = info['email'] if info['email']
    primary_user.first_name = info['first_name'] if info['first_name']
    primary_user.last_name = info['last_name'] if info['last_name']

    if (!primary_user.email or primary_user.email.empty?) and (!primary_user.identity_url or primary_user.identity_url.empty?)
      raise "Must have supply at least one of 'email' or 'identity_url' to User.register"
    end

    act_as_system_user do
      primary_user.save!
    end

    primary_user
  end

  protected

  def change_all_uuid_refs(old_uuid:, new_uuid:)
    ActiveRecord::Base.descendants.reject(&:abstract_class?).each do |klass|
      klass.columns.each do |col|
        if col.name.end_with?('_uuid')
          column = col.name.to_sym
          klass.where(column => old_uuid).update_all(column => new_uuid)
        end
      end
    end
  end

  def ensure_ownership_path_leads_to_user
    true
  end

  def permission_to_update
    if username_changed? || redirect_to_user_uuid_changed? || email_changed?
      current_user.andand.is_admin
    else
      # users must be able to update themselves (even if they are
      # inactive) in order to create sessions
      self == current_user or super
    end
  end

  def permission_to_create
    current_user.andand.is_admin or
      (self == current_user &&
       self.redirect_to_user_uuid.nil? &&
       self.is_active == Rails.configuration.Users.NewUsersAreActive)
  end

  def check_auto_admin
    return if self.uuid.end_with?('anonymouspublic')
    if (User.where("email = ?",self.email).where(:is_admin => true).count == 0 and
        !Rails.configuration.Users.AutoAdminUserWithEmail.empty? and self.email == Rails.configuration.Users["AutoAdminUserWithEmail"]) or
       (User.where("uuid not like '%-000000000000000'").where(:is_admin => true).count == 0 and
        Rails.configuration.Users.AutoAdminFirstUser)
      self.is_admin = true
      self.is_active = true
    end
  end

  def find_usable_username_from(basename)
    # If "basename" is a usable username, return that.
    # Otherwise, find a unique username "basenameN", where N is the
    # smallest integer greater than 1, and return that.
    # Return nil if a unique username can't be found after reasonable
    # searching.
    quoted_name = self.class.connection.quote_string(basename)
    next_username = basename
    next_suffix = 1
    while Rails.configuration.Users.AutoSetupUsernameBlacklist[next_username]
      next_suffix += 1
      next_username = "%s%i" % [basename, next_suffix]
    end
    0.upto(6).each do |suffix_len|
      pattern = "%s%s" % [quoted_name, "_" * suffix_len]
      self.class.unscoped.
          where("username like '#{pattern}'").
          select(:username).
          order('username asc').
          each do |other_user|
        if other_user.username > next_username
          break
        elsif other_user.username == next_username
          next_suffix += 1
          next_username = "%s%i" % [basename, next_suffix]
        end
      end
      return next_username if (next_username.size <= pattern.size)
    end
    nil
  end

  def prevent_privilege_escalation
    if current_user.andand.is_admin
      return true
    end
    if self.is_active_changed?
      if self.is_active != self.is_active_was
        logger.warn "User #{current_user.uuid} tried to change is_active from #{self.is_active_was} to #{self.is_active} for #{self.uuid}"
        self.is_active = self.is_active_was
      end
    end
    if self.is_admin_changed?
      if self.is_admin != self.is_admin_was
        logger.warn "User #{current_user.uuid} tried to change is_admin from #{self.is_admin_was} to #{self.is_admin} for #{self.uuid}"
        self.is_admin = self.is_admin_was
      end
    end
    true
  end

  def prevent_inactive_admin
    if self.is_admin and not self.is_active
      # There is no known use case for the strange set of permissions
      # that would result from this change. It's safest to assume it's
      # a mistake and disallow it outright.
      raise "Admin users cannot be inactive"
    end
    true
  end

  def search_permissions(start, graph, merged={}, upstream_mask=nil, upstream_path={})
    nextpaths = graph[start]
    return merged if !nextpaths
    return merged if upstream_path.has_key? start
    upstream_path[start] = true
    upstream_mask ||= ALL_PERMISSIONS
    nextpaths.each do |head, mask|
      merged[head] ||= {}
      mask.each do |k,v|
        merged[head][k] ||= v if upstream_mask[k]
      end
      search_permissions(head, graph, merged, upstream_mask.select { |k,v| v && merged[head][k] }, upstream_path)
    end
    upstream_path.delete start
    merged
  end

  def create_user_repo_link(repo_name)
    # repo_name is optional
    if not repo_name
      logger.warn ("Repository name not given for #{self.uuid}.")
      return
    end

    repo = Repository.where(owner_uuid: uuid, name: repo_name).first_or_create!
    logger.info { "repo uuid: " + repo[:uuid] }
    repo_perm = Link.where(tail_uuid: uuid, head_uuid: repo.uuid,
                           link_class: "permission",
                           name: "can_manage").first_or_create!
    logger.info { "repo permission: " + repo_perm[:uuid] }
    return repo_perm
  end

  # create login permission for the given vm_uuid, if it does not already exist
  def create_vm_login_permission_link(vm_uuid, repo_name)
    # vm uuid is optional
    return if vm_uuid == ""

    vm = VirtualMachine.where(uuid: vm_uuid).first
    if !vm
      logger.warn "Could not find virtual machine for #{vm_uuid.inspect}"
      raise "No vm found for #{vm_uuid}"
    end

    logger.info { "vm uuid: " + vm[:uuid] }
    login_attrs = {
      tail_uuid: uuid, head_uuid: vm.uuid,
      link_class: "permission", name: "can_login",
    }

    login_perm = Link.
      where(login_attrs).
      select { |link| link.properties["username"] == repo_name }.
      first

    login_perm ||= Link.
      create(login_attrs.merge(properties: {"username" => repo_name}))

    logger.info { "login permission: " + login_perm[:uuid] }
    login_perm
  end

  # add the user to the 'All users' group
  def create_user_group_link
    return (Link.where(tail_uuid: self.uuid,
                       head_uuid: all_users_group_uuid,
                       link_class: 'permission',
                       name: 'can_read').first or
            Link.create(tail_uuid: self.uuid,
                        head_uuid: all_users_group_uuid,
                        link_class: 'permission',
                        name: 'can_read'))
  end

  # Give the special "System group" permission to manage this user and
  # all of this user's stuff.
  def add_system_group_permission_link
    return true if uuid == system_user_uuid
    act_as_system_user do
      Link.create(link_class: 'permission',
                  name: 'can_manage',
                  tail_uuid: system_group_uuid,
                  head_uuid: self.uuid)
    end
  end

  # Send admin notifications
  def send_admin_notifications
    AdminNotifier.new_user(self).deliver_now
    if not self.is_active then
      AdminNotifier.new_inactive_user(self).deliver_now
    end
  end

  # Automatically setup if is_active flag turns on
  def setup_on_activate
    return if [system_user_uuid, anonymous_user_uuid].include?(self.uuid)
    if is_active &&
      (new_record? || saved_change_to_is_active? || will_save_change_to_is_active?)
      setup
    end
  end

  # Automatically setup new user during creation
  def auto_setup_new_user
    setup
  end

  # Send notification if the user saved profile for the first time
  def send_profile_created_notification
    if saved_change_to_prefs?
      if prefs_before_last_save.andand.empty? || !prefs_before_last_save.andand['profile']
        profile_notification_address = Rails.configuration.Users.UserProfileNotificationAddress
        ProfileNotifier.profile_created(self, profile_notification_address).deliver_now if profile_notification_address and !profile_notification_address.empty?
      end
    end
  end

  def verify_repositories_empty
    unless repositories.first.nil?
      errors.add(:username, "can't be unset when the user owns repositories")
      throw(:abort)
    end
  end

  def sync_repository_names
    old_name_re = /^#{Regexp.escape(username_before_last_save)}\//
    name_sub = "#{username}/"
    repositories.find_each do |repo|
      repo.name = repo.name.sub(old_name_re, name_sub)
      repo.save!
    end
  end
end
