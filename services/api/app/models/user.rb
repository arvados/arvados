class User < OrvosModel
  include AssignUuid
  include KindAndEtag
  include CommonApiTemplate
  serialize :prefs, Hash
  has_many :api_client_authorizations
  before_update :prevent_privilege_escalation

  api_accessible :superuser, :extend => :common do |t|
    t.add :email
    t.add :full_name
    t.add :first_name
    t.add :last_name
    t.add :identity_url
    t.add :is_admin
    t.add :prefs
  end

  ALL_PERMISSIONS = {read: true, write: true, manage: true}

  def full_name
    "#{first_name} #{last_name}"
  end

  def groups_i_can(verb)
    self.group_permissions.select { |uuid, mask| mask[verb] }.keys
  end

  def can?(actions)
    actions.each do |action, target|
      target_uuid = target
      if target.respond_to? :uuid
        target_uuid = target.uuid
      end
      next if target_uuid == self.uuid
      next if (group_permissions[target_uuid] and
               group_permissions[target_uuid][action])
      if target.respond_to? :owner
        next if target.owner == self.uuid
        next if (group_permissions[target.owner] and
                 group_permissions[target.owner][action])
      end
      return false
    end
    true
  end

  def self.invalidate_permissions_cache
    Rails.cache.delete_matched(/^groups_for_user_/)
  end

  protected

  def permission_to_create
    Thread.current[:user] == self or
      (Thread.current[:user] and Thread.current[:user].is_admin)
  end

  def prevent_privilege_escalation
    if self.is_admin_changed? and !current_user.is_admin
      if current_user.uuid == self.uuid
        if self.is_admin != self.is_admin_was
          logger.warn "User #{self.uuid} tried to change is_admin from #{self.is_admin_was} to #{self.is_admin}"
          self.is_admin = self.is_admin_was
        end
      end
    end
    true
  end

  def group_permissions
    Rails.cache.fetch "groups_for_user_#{self.uuid}" do
      permissions_from = {}
      todo = {self.uuid => true}
      done = {}
      while !todo.empty?
        lookup_uuids = todo.keys
        lookup_uuids.each do |uuid| done[uuid] = true end
        todo = {}
        Link.where('tail_uuid in (?) and link_class = ? and head_kind = ?',
                   lookup_uuids,
                   'permission',
                   'orvos#group').each do |link|
          unless done.has_key? link.head_uuid
            todo[link.head_uuid] = true
          end
          link_permissions = {}
          case link.name
          when 'can_read'
            link_permissions = {read:true}
          when 'can_write'
            link_permissions = {read:true,write:true}
          when 'can_manage'
            link_permissions = ALL_PERMISSIONS
          end
          permissions_from[link.tail_uuid] ||= {}
          permissions_from[link.tail_uuid][link.head_uuid] ||= {}
          link_permissions.each do |k,v|
            permissions_from[link.tail_uuid][link.head_uuid][k] ||= v
          end
        end
      end
      search_permissions(self.uuid, permissions_from)
    end
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
