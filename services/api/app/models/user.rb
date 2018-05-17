# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

require 'can_be_an_owner'
require 'refresh_permission_view'

class User < ArvadosModel
  include HasUuid
  include KindAndEtag
  include CommonApiTemplate
  include CanBeAnOwner

  serialize :prefs, Hash
  has_many :api_client_authorizations
  validates(:username,
            format: {
              with: /\A[A-Za-z][A-Za-z0-9]*\z/,
              message: "must begin with a letter and contain only alphanumerics",
            },
            uniqueness: true,
            allow_nil: true)
  before_update :prevent_privilege_escalation
  before_update :prevent_inactive_admin
  before_update :verify_repositories_empty, :if => Proc.new { |user|
    user.username.nil? and user.username_changed?
  }
  before_update :setup_on_activate
  before_create :check_auto_admin
  before_create :set_initial_username, :if => Proc.new { |user|
    user.username.nil? and user.email
  }
  after_create :add_system_group_permission_link
  after_create :invalidate_permissions_cache
  after_create :auto_setup_new_user, :if => Proc.new { |user|
    Rails.configuration.auto_setup_new_users and
    (user.uuid != system_user_uuid) and
    (user.uuid != anonymous_user_uuid)
  }
  after_create :send_admin_notifications
  after_update :send_profile_created_notification
  after_update :sync_repository_names, :if => Proc.new { |user|
    (user.uuid != system_user_uuid) and
    user.username_changed? and
    (not user.username_was.nil?)
  }

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

  def full_name
    "#{first_name} #{last_name}".strip
  end

  def is_invited
    !!(self.is_active ||
       Rails.configuration.new_users_are_active ||
       self.groups_i_can(:read).select { |x| x.match(/-f+$/) }.first)
  end

  def groups_i_can(verb)
    my_groups = self.group_permissions.select { |uuid, mask| mask[verb] }.keys
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
      next if (group_permissions[target_uuid] and
               group_permissions[target_uuid][action])
      if target.respond_to? :owner_uuid
        next if target.owner_uuid == self.uuid
        next if (group_permissions[target.owner_uuid] and
                 group_permissions[target.owner_uuid][action])
      end
      sufficient_perms = case action
                         when :manage
                           ['can_manage']
                         when :write
                           ['can_manage', 'can_write']
                         when :read
                           ['can_manage', 'can_write', 'can_read']
                         else
                           # (Skip this kind of permission opportunity
                           # if action is an unknown permission type)
                         end
      if sufficient_perms
        # Check permission links with head_uuid pointing directly at
        # the target object. If target is a Group, this is redundant
        # and will fail except [a] if permission caching is broken or
        # [b] during a race condition, where a permission link has
        # *just* been added.
        if Link.where(link_class: 'permission',
                      name: sufficient_perms,
                      tail_uuid: groups_i_can(action) + [self.uuid],
                      head_uuid: target_uuid).any?
          next
        end
      end
      return false
    end
    true
  end

  def self.invalidate_permissions_cache(timestamp=nil)
    if Rails.configuration.async_permissions_update
      timestamp = DbCurrentTime::db_current_time.to_i if timestamp.nil?
      connection.execute "NOTIFY invalidate_permissions_cache, '#{timestamp}'"
    else
      refresh_permission_view
    end
  end

  def invalidate_permissions_cache(timestamp=nil)
    User.invalidate_permissions_cache
  end

  # Return a hash of {user_uuid: group_perms}
  def self.all_group_permissions
    all_perms = {}
    ActiveRecord::Base.connection.
      exec_query("SELECT user_uuid, target_owner_uuid, perm_level, trashed
                  FROM #{PERMISSION_VIEW}
                  WHERE target_owner_uuid IS NOT NULL",
                  # "name" arg is a query label that appears in logs:
                  "all_group_permissions",
                  ).rows.each do |user_uuid, group_uuid, max_p_val, trashed|
      all_perms[user_uuid] ||= {}
      all_perms[user_uuid][group_uuid] = PERMS_FOR_VAL[max_p_val.to_i]
    end
    all_perms
  end

  # Return a hash of {group_uuid: perm_hash} where perm_hash[:read]
  # and perm_hash[:write] are true if this user can read and write
  # objects owned by group_uuid.
  def group_permissions
    group_perms = {self.uuid => {:read => true, :write => true, :manage => true}}
    ActiveRecord::Base.connection.
      exec_query("SELECT target_owner_uuid, perm_level, trashed
                  FROM #{PERMISSION_VIEW}
                  WHERE user_uuid = $1
                  AND target_owner_uuid IS NOT NULL",
                  # "name" arg is a query label that appears in logs:
                  "group_permissions for #{uuid}",
                  # "binds" arg is an array of [col_id, value] for '$1' vars:
                  [[nil, uuid]],
                ).rows.each do |group_uuid, max_p_val, trashed|
      group_perms[group_uuid] = PERMS_FOR_VAL[max_p_val.to_i]
    end
    group_perms
  end

  # create links
  def setup(openid_prefix:, repo_name: nil, vm_uuid: nil)
    oid_login_perm = create_oid_login_perm openid_prefix
    repo_perm = create_user_repo_link repo_name
    vm_login_perm = create_vm_login_permission_link(vm_uuid, username) if vm_uuid
    group_perm = create_user_group_link

    return [oid_login_perm, repo_perm, vm_login_perm, group_perm, self].compact
  end

  # delete user signatures, login, repo, and vm perms, and mark as inactive
  def unsetup
    # delete oid_login_perms for this user
    Link.destroy_all(tail_uuid: self.email,
                     link_class: 'permission',
                     name: 'can_login')

    # delete repo_perms for this user
    Link.destroy_all(tail_uuid: self.uuid,
                     link_class: 'permission',
                     name: 'can_manage')

    # delete vm_login_perms for this user
    Link.destroy_all(tail_uuid: self.uuid,
                     link_class: 'permission',
                     name: 'can_login')

    # delete "All users" group read permissions for this user
    group = Group.where(name: 'All users').select do |g|
      g[:uuid].match(/-f+$/)
    end.first
    Link.destroy_all(tail_uuid: self.uuid,
                     head_uuid: group[:uuid],
                     link_class: 'permission',
                     name: 'can_read')

    # delete any signatures by this user
    Link.destroy_all(link_class: 'signature',
                     tail_uuid: self.uuid)

    # delete user preferences (including profile)
    self.prefs = {}

    # mark the user as inactive
    self.is_active = false
    self.save!
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

  def update_uuid(new_uuid:)
    if !current_user.andand.is_admin
      raise PermissionDeniedError
    end
    if uuid == system_user_uuid || uuid == anonymous_user_uuid
      raise "update_uuid cannot update system accounts"
    end
    if self.class != self.class.resource_class_for_uuid(new_uuid)
      raise "invalid new_uuid #{new_uuid.inspect}"
    end
    transaction(requires_new: true) do
      reload
      old_uuid = self.uuid
      self.uuid = new_uuid
      save!(validate: false)
      change_all_uuid_refs(old_uuid: old_uuid, new_uuid: new_uuid)
    end
  end

  # Move this user's (i.e., self's) owned items into new_owner_uuid.
  # Also redirect future uses of this account to
  # redirect_to_user_uuid, i.e., when a caller authenticates to this
  # account in the future, the account redirect_to_user_uuid account
  # will be used instead.
  #
  # current_user must have admin privileges, i.e., the caller is
  # responsible for checking permission to do this.
  def merge(new_owner_uuid:, redirect_to_user_uuid:)
    raise PermissionDeniedError if !current_user.andand.is_admin
    raise "not implemented" if !redirect_to_user_uuid
    transaction(requires_new: true) do
      reload
      raise "cannot merge an already merged user" if self.redirect_to_user_uuid

      new_user = User.where(uuid: redirect_to_user_uuid).first
      raise "user does not exist" if !new_user
      raise "cannot merge to an already merged user" if new_user.redirect_to_user_uuid

      # Existing API tokens are updated to authenticate to the new
      # user.
      ApiClientAuthorization.
        where(user_id: id).
        update_all(user_id: new_user.id)

      # References to the old user UUID in the context of a user ID
      # (rather than a "home project" in the project hierarchy) are
      # updated to point to the new user.
      [
        [AuthorizedKey, :owner_uuid],
        [AuthorizedKey, :authorized_user_uuid],
        [Repository, :owner_uuid],
        [Link, :owner_uuid],
        [Link, :tail_uuid],
        [Link, :head_uuid],
      ].each do |klass, column|
        klass.where(column => uuid).update_all(column => new_user.uuid)
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

      update_attributes!(redirect_to_user_uuid: new_user.uuid)
      invalidate_permissions_cache
    end
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
    if username_changed? || redirect_to_user_uuid_changed?
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
       self.is_active == Rails.configuration.new_users_are_active)
  end

  def check_auto_admin
    return if self.uuid.end_with?('anonymouspublic')
    if (User.where("email = ?",self.email).where(:is_admin => true).count == 0 and
        Rails.configuration.auto_admin_user and self.email == Rails.configuration.auto_admin_user) or
       (User.where("uuid not like '%-000000000000000'").where(:is_admin => true).count == 0 and
        Rails.configuration.auto_admin_first_user)
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
    while Rails.configuration.auto_setup_name_blacklist.include?(next_username)
      next_suffix += 1
      next_username = "%s%i" % [basename, next_suffix]
    end
    0.upto(6).each do |suffix_len|
      pattern = "%s%s" % [quoted_name, "_" * suffix_len]
      self.class.
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

  def create_oid_login_perm(openid_prefix)
    # Check oid_login_perm
    oid_login_perms = Link.where(tail_uuid: self.email,
                                 head_uuid: self.uuid,
                                 link_class: 'permission',
                                 name: 'can_login')

    if !oid_login_perms.any?
      # create openid login permission
      oid_login_perm = Link.create(link_class: 'permission',
                                   name: 'can_login',
                                   tail_uuid: self.email,
                                   head_uuid: self.uuid,
                                   properties: {
                                     "identity_url_prefix" => openid_prefix,
                                   })
      logger.info { "openid login permission: " + oid_login_perm[:uuid] }
    else
      oid_login_perm = oid_login_perms.first
    end

    return oid_login_perm
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
    return if !vm_uuid

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
                       head_uuid: all_users_group[:uuid],
                       link_class: 'permission',
                       name: 'can_read').first or
            Link.create(tail_uuid: self.uuid,
                        head_uuid: all_users_group[:uuid],
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
    if is_active && (new_record? || is_active_changed?)
      setup(openid_prefix: Rails.configuration.default_openid_prefix)
    end
  end

  # Automatically setup new user during creation
  def auto_setup_new_user
    setup(openid_prefix: Rails.configuration.default_openid_prefix)
    if username
      create_vm_login_permission_link(Rails.configuration.auto_setup_new_users_with_vm_uuid,
                                      username)
      repo_name = "#{username}/#{username}"
      if Rails.configuration.auto_setup_new_users_with_repository and
          Repository.where(name: repo_name).first.nil?
        repo = Repository.create!(name: repo_name, owner_uuid: uuid)
        Link.create!(tail_uuid: uuid, head_uuid: repo.uuid,
                     link_class: "permission", name: "can_manage")
      end
    end
  end

  # Send notification if the user saved profile for the first time
  def send_profile_created_notification
    if self.prefs_changed?
      if self.prefs_was.andand.empty? || !self.prefs_was.andand['profile']
        profile_notification_address = Rails.configuration.user_profile_notification_address
        ProfileNotifier.profile_created(self, profile_notification_address).deliver_now if profile_notification_address
      end
    end
  end

  def verify_repositories_empty
    unless repositories.first.nil?
      errors.add(:username, "can't be unset when the user owns repositories")
      false
    end
  end

  def sync_repository_names
    old_name_re = /^#{Regexp.escape(username_was)}\//
    name_sub = "#{username}/"
    repositories.find_each do |repo|
      repo.name = repo.name.sub(old_name_re, name_sub)
      repo.save!
    end
  end
end
