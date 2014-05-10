# Protect 

module CanBeAnOwner

  def self.included(base)
    # Rails' "has_many" can prevent us from destroying the owner
    # record when other objects refer to it.
    ActiveRecord::Base.connection.tables.each do |t|
      next if t == base.table_name
      next if t == 'schema_migrations'
      klass = t.classify.constantize
      next unless klass and 'owner_uuid'.in?(klass.columns.collect(&:name))
      base.has_many(t.to_sym,
                    foreign_key: :owner_uuid,
                    primary_key: :uuid,
                    dependent: :restrict)
    end
    # We need custom protection for changing an owner's primary
    # key. (Apart from this restriction, admins are allowed to change
    # UUIDs.)
    base.validate :restrict_uuid_change_breaking_associations
  end

  protected

  def restrict_uuid_change_breaking_associations
    return true if new_record? or not uuid_changed?

    # Check for objects that have my old uuid listed as their owner.
    self.class.reflect_on_all_associations(:has_many).each do |assoc|
      next unless assoc.foreign_key == :owner_uuid
      if assoc.klass.where(owner_uuid: uuid_was).any?
        errors.add(:uuid,
                   "cannot be changed on a #{self.class} that owns objects")
        return false
      end
    end

    # if I owned myself before, I'll just continue to own myself with
    # my new uuid.
    if owner_uuid == uuid_was
      self.owner_uuid = uuid
    end
  end

end
