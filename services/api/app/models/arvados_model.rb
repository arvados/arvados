class ArvadosModel < ActiveRecord::Base
  self.abstract_class = true

  include CurrentApiClient      # current_user, current_api_client, etc.

  attr_protected :created_at
  attr_protected :modified_by_user
  attr_protected :modified_by_client
  attr_protected :modified_at
  before_create :ensure_permission_to_create
  before_update :ensure_permission_to_update
  before_create :update_modified_by_fields
  before_update :maybe_update_modified_by_fields
  validate :ensure_serialized_attribute_type

  def self.kind_class(kind)
    kind.match(/^arvados\#(.+?)(_list|List)?$/)[1].pluralize.classify.constantize rescue nil
  end

  def eager_load_associations
    self.class.columns.each do |col|
      re = col.name.match /^(.*)_kind$/
      if (re and
          self.respond_to? re[1].to_sym and
          (auuid = self.send((re[1] + '_uuid').to_sym)) and
          (aclass = self.class.kind_class(self.send(col.name.to_sym))) and
          (aobject = aclass.where('uuid=?', auuid).first))
        self.instance_variable_set('@'+re[1], aobject)
      end
    end
  end

  protected

  def ensure_permission_to_create
    raise "Permission denied" unless permission_to_create
  end

  def permission_to_create
    current_user
  end

  def ensure_permission_to_update
    raise "Permission denied" unless permission_to_update
  end

  def permission_to_update
    if !current_user
      logger.warn "Anonymous user tried to update #{self.class.to_s} #{self.uuid_was}"
      return false
    end
    if self.uuid_changed?
      logger.warn "User #{current_user.uuid} tried to change uuid of #{self.class.to_s} #{self.uuid_was} to #{self.uuid}"
      return false
    end
    return true if current_user.is_admin
    if self.owner_changed?
      if current_user.uuid == self.owner or
          current_user.can? write: self.owner
        # current_user is, or has :write permission on, the new owner
      else
        logger.warn "User #{current_user.uuid} tried to change owner of #{self.class.to_s} #{self.uuid} to #{self.owner} but does not have permission to write to #{self.owner}"
        return false
      end
    end
    if current_user.uuid == self.owner_was or
        current_user.uuid == self.uuid or
        current_user.can? write: self.owner_was
      # current user is, or has :write permission on, the previous owner
      return true
    else
      logger.warn "User #{current_user.uuid} tried to modify #{self.class.to_s} #{self.uuid} but does not have permission to write #{self.owner_was}"
      return false
    end
  end

  def maybe_update_modified_by_fields
    update_modified_by_fields if self.changed?
  end

  def update_modified_by_fields
    self.created_at ||= Time.now
    self.owner ||= current_default_owner
    self.modified_at = Time.now
    self.modified_by_user = current_user ? current_user.uuid : nil
    self.modified_by_client = current_api_client ? current_api_client.uuid : nil
  end

  def ensure_serialized_attribute_type
    # Specifying a type in the "serialize" declaration causes rails to
    # raise an exception if a different data type is retrieved from
    # the database during load().  The validation preventing such
    # crash-inducing records from being inserted in the database in
    # the first place seems to have been left as an exercise to the
    # developer.
    self.class.serialized_attributes.each do |colname, attr|
      if attr.object_class
        unless self.attributes[colname].is_a? attr.object_class
          self.errors.add colname.to_sym, "must be a #{attr.object_class.to_s}"
        end
      end
    end
  end
end
