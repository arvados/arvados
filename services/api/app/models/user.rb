class User < ArvadosModel
  include AssignUuid
  include KindAndEtag
  include CommonApiTemplate
  serialize :prefs, Hash
  has_many :api_client_authorizations
  before_update :prevent_privilege_escalation
  before_update :prevent_inactive_admin
  before_create :check_auto_admin
  after_create AdminNotifier

  has_many :authorized_keys, :foreign_key => :authorized_user_uuid, :primary_key => :uuid

  api_accessible :user, extend: :common do |t|
    t.add :email
    t.add :full_name
    t.add :first_name
    t.add :last_name
    t.add :identity_url
    t.add :is_active
    t.add :is_admin
    t.add :is_invited
    t.add :prefs
  end

  ALL_PERMISSIONS = {read: true, write: true, manage: true}

  def full_name
    "#{first_name} #{last_name}"
  end

  def is_invited
    !!(self.is_active ||
       Rails.configuration.new_users_are_active ||
       self.groups_i_can(:read).select { |x| x.match /-f+$/ }.first)
  end

  def groups_i_can(verb)
    self.group_permissions.select { |uuid, mask| mask[verb] }.keys
  end

  def can?(actions)
    return true if is_admin
    actions.each do |action, target|
      target_uuid = target
      if target.respond_to? :uuid
        target_uuid = target.uuid
      end
      next if target_uuid == self.uuid
      next if (group_permissions[target_uuid] and
               group_permissions[target_uuid][action])
      if target.respond_to? :owner_uuid
        next if target.owner_uuid == self.uuid
        next if (group_permissions[target.owner_uuid] and
                 group_permissions[target.owner_uuid][action])
      end
      return false
    end
    true
  end

  def self.invalidate_permissions_cache
    Rails.cache.delete_matched(/^groups_for_user_/)
  end

  # Return a hash of {group_uuid: perm_hash} where perm_hash[:read]
  # and perm_hash[:write] are true if this user can read and write
  # objects owned by group_uuid.
  def group_permissions
    Rails.cache.fetch "groups_for_user_#{self.uuid}" do
      permissions_from = {}
      todo = {self.uuid => true}
      done = {}
      while !todo.empty?
        lookup_uuids = todo.keys
        lookup_uuids.each do |uuid| done[uuid] = true end
        todo = {}
        newgroups = []
        Group.where('owner_uuid in (?)', lookup_uuids).each do |group|
          newgroups << [group.owner_uuid, group.uuid, 'can_manage']
        end
        Link.where('tail_uuid in (?) and link_class = ? and head_kind = ?',
                   lookup_uuids,
                   'permission',
                   'arvados#group').each do |link|
          newgroups << [link.tail_uuid, link.head_uuid, link.name]
        end
        newgroups.each do |tail_uuid, head_uuid, perm_name|
          unless done.has_key? head_uuid
            todo[head_uuid] = true
          end
          link_permissions = {}
          case perm_name
          when 'can_read'
            link_permissions = {read:true}
          when 'can_write'
            link_permissions = {read:true,write:true}
          when 'can_manage'
            link_permissions = ALL_PERMISSIONS
          end
          permissions_from[tail_uuid] ||= {}
          permissions_from[tail_uuid][head_uuid] ||= {}
          link_permissions.each do |k,v|
            permissions_from[tail_uuid][head_uuid][k] ||= v
          end
        end
      end
      search_permissions(self.uuid, permissions_from)
    end
  end

  protected

  def permission_to_update
    # users must be able to update themselves (even if they are
    # inactive) in order to create sessions
    self == current_user or super
  end

  def permission_to_create
    current_user.andand.is_admin or
      (self == current_user and
       self.is_active == Rails.configuration.new_users_are_active)
  end

  def check_auto_admin
    if User.where("uuid not like '%-000000000000000'").where(:is_admin => true).count == 0 and not Rails.configuration.auto_admin_user.nil?
      if current_user.email == Rails.configuration.auto_admin_user
        self.is_admin = true
        self.is_active = true
      end
    end
  end

  def prevent_privilege_escalation
    if current_user.andand.is_admin
      return true
    end
    if self.is_active_changed?
      if self.is_active != self.is_active_was
        logger.warn "User #{current_user.uuid} tried to change is_active from #{self.is_admin_was} to #{self.is_admin} for #{self.uuid}"
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
end
