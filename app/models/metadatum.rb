class Metadatum < OrvosModel
  include AssignUuid
  include KindAndEtag
  include CommonApiTemplate
  serialize :info, Hash
  before_create :permission_to_attach_to_objects
  before_update :permission_to_attach_to_objects

  api_accessible :superuser, :extend => :common do |t|
    t.add :tail_kind
    t.add :tail
    t.add :metadata_class
    t.add :name
    t.add :head_kind
    t.add :head
    t.add :info
  end

  def info
    @info ||= Hash.new
    super
  end

  protected

  def permission_to_attach_to_objects
    # Anonymous users cannot write metadata
    return false if !current_user

    # All users can write metadata that doesn't affect permissions
    return true if self.metadata_class != 'permission'

    # Administrators can grant permissions
    return true if current_user.is_admin

    # All users can grant permissions on objects they created themselves
    head_obj = self.class.
      kind_class(self.head_kind).
      where('uuid=?',head_uuid).
      first
    if head_obj
      return true if head_obj.created_by_user == current_user.uuid
    end

    # Users with "can_manage" permission on an object can grant
    # permissions on that object
    has_manage_permission = self.class.
      where('metadata_class=? AND name=? AND tail=? AND head=?',
            'permission', 'can_manage', current_user.uuid, self.head).
      count > 0
    return true if has_manage_permission

    # Default = deny.
    false
  end
end
