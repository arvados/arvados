require 'can_be_an_owner'

class Group < ArvadosModel
  include HasUuid
  include KindAndEtag
  include CommonApiTemplate
  include CanBeAnOwner
  after_create :invalidate_permissions_cache
  after_update :maybe_invalidate_permissions_cache
  before_create :assign_name

  api_accessible :user, extend: :common do |t|
    t.add :name
    t.add :group_class
    t.add :description
    t.add :writable_by
  end

  def maybe_invalidate_permissions_cache
    if uuid_changed? or owner_uuid_changed?
      # This can change users' permissions on other groups as well as
      # this one.
      invalidate_permissions_cache
    end
  end

  def invalidate_permissions_cache
    # Ensure a new group can be accessed by the appropriate users
    # immediately after being created.
    User.invalidate_permissions_cache
  end

  def assign_name
    if self.new_record? and (self.name.nil? or self.name.empty?)
      self.name = self.uuid
    end
    true
  end

end
