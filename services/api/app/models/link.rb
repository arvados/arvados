class Link < ArvadosModel
  include HasUuid
  include KindAndEtag
  include CommonApiTemplate
  serialize :properties, Hash
  before_create :permission_to_attach_to_objects
  before_update :permission_to_attach_to_objects
  after_update :maybe_invalidate_permissions_cache
  after_create :maybe_invalidate_permissions_cache
  after_destroy :maybe_invalidate_permissions_cache
  attr_accessor :head_kind, :tail_kind
  validate :name_links_are_obsolete

  api_accessible :user, extend: :common do |t|
    t.add :tail_uuid
    t.add :link_class
    t.add :name
    t.add :head_uuid
    t.add :head_kind
    t.add :tail_kind
    t.add :properties
  end

  def properties
    @properties ||= Hash.new
    super
  end

  def head_kind
    if k = ArvadosModel::resource_class_for_uuid(head_uuid)
      k.kind
    end
  end

  def tail_kind
    if k = ArvadosModel::resource_class_for_uuid(tail_uuid)
      k.kind
    end
  end

  protected

  def permission_to_attach_to_objects
    # Anonymous users cannot write links
    return false if !current_user

    # All users can write links that don't affect permissions
    return true if self.link_class != 'permission'

    # Administrators can grant permissions
    return true if current_user.is_admin

    # All users can grant permissions on objects they own or can manage
    head_obj = ArvadosModel.find_by_uuid(head_uuid)
    return true if current_user.can?(manage: head_obj)

    # Default = deny.
    false
  end

  def maybe_invalidate_permissions_cache
    if self.link_class == 'permission'
      # Clearing the entire permissions cache can generate many
      # unnecessary queries if many active users are not affected by
      # this change. In such cases it would be better to search cached
      # permissions for head_uuid and tail_uuid, and invalidate the
      # cache for only those users. (This would require a browseable
      # cache.)
      User.invalidate_permissions_cache
    end
  end

  def name_links_are_obsolete
    if link_class == 'name'
      errors.add('name', 'Name links are obsolete')
      false
    else
      true
    end
  end

  # A user is permitted to create, update or modify a permission link
  # if and only if they have "manage" permission on the object
  # indicated by the permission link's head_uuid.
  #
  # All other links are treated as regular ArvadosModel objects.
  #
  def ensure_owner_uuid_is_permitted
    if link_class == 'permission'
      ob = ArvadosModel.find_by_uuid(head_uuid)
      raise PermissionDeniedError unless current_user.can?(manage: ob)
      # All permission links should be owned by the system user.
      self.owner_uuid = system_user_uuid
      return true
    else
      super
    end
  end

end
