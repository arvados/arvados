require 'assign_uuid'
class ArvadosModel < ActiveRecord::Base
  self.abstract_class = true

  include CurrentApiClient      # current_user, current_api_client, etc.

  attr_protected :created_at
  attr_protected :modified_by_user_uuid
  attr_protected :modified_by_client_uuid
  attr_protected :modified_at
  after_initialize :log_start_state
  before_create :ensure_permission_to_create
  before_update :ensure_permission_to_update
  before_destroy :ensure_permission_to_destroy
  before_create :update_modified_by_fields
  before_update :maybe_update_modified_by_fields
  after_create :log_create
  after_update :log_update
  after_destroy :log_destroy
  validate :ensure_serialized_attribute_type
  validate :normalize_collection_uuids

  has_many :permissions, :foreign_key => :head_uuid, :class_name => 'Link', :primary_key => :uuid, :conditions => "link_class = 'permission'"

  class PermissionDeniedError < StandardError
    def http_status
      403
    end
  end

  class UnauthorizedError < StandardError
    def http_status
      401
    end
  end

  def self.kind_class(kind)
    kind.match(/^arvados\#(.+?)(_list|List)?$/)[1].pluralize.classify.constantize rescue nil
  end

  def href
    "#{current_api_base}/#{self.class.to_s.pluralize.underscore}/#{self.uuid}"
  end

  def self.searchable_columns operator
    textonly_operator = !operator.match(/[<=>]/)
    self.columns.collect do |col|
      if col.name == 'owner_uuid'
        nil
      elsif [:string, :text].index(col.type)
        col.name
      elsif !textonly_operator and [:datetime, :integer].index(col.type)
        col.name
      end
    end.compact
  end

  def self.attribute_column attr
    self.columns.select { |col| col.name == attr.to_s }.first
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

  def self.readable_by user
    uuid_list = [user.uuid, *user.groups_i_can(:read)]
    sanitized_uuid_list = uuid_list.
      collect { |uuid| sanitize(uuid) }.join(', ')
    or_references_me = ''
    if self == Link and user
      or_references_me = "OR (#{table_name}.link_class in (#{sanitize 'permission'}, #{sanitize 'resources'}) AND #{sanitize user.uuid} IN (#{table_name}.head_uuid, #{table_name}.tail_uuid))"
    end
    joins("LEFT JOIN links permissions ON permissions.head_uuid in (#{table_name}.owner_uuid, #{table_name}.uuid) AND permissions.tail_uuid in (#{sanitized_uuid_list}) AND permissions.link_class='permission'").
      where("?=? OR #{table_name}.owner_uuid in (?) OR #{table_name}.uuid=? OR permissions.head_uuid IS NOT NULL #{or_references_me}",
            true, user.is_admin,
            uuid_list,
            user.uuid)
  end

  def logged_attributes
    attributes
  end

  protected

  def ensure_permission_to_create
    raise PermissionDeniedError unless permission_to_create
  end

  def permission_to_create
    current_user.andand.is_active
  end

  def ensure_permission_to_update
    raise PermissionDeniedError unless permission_to_update
  end

  def permission_to_update
    if !current_user
      logger.warn "Anonymous user tried to update #{self.class.to_s} #{self.uuid_was}"
      return false
    end
    if !current_user.is_active
      logger.warn "Inactive user #{current_user.uuid} tried to update #{self.class.to_s} #{self.uuid_was}"
      return false
    end
    return true if current_user.is_admin
    if self.uuid_changed?
      logger.warn "User #{current_user.uuid} tried to change uuid of #{self.class.to_s} #{self.uuid_was} to #{self.uuid}"
      return false
    end
    if self.owner_uuid_changed?
      if current_user.uuid == self.owner_uuid or
          current_user.can? write: self.owner_uuid
        # current_user is, or has :write permission on, the new owner
      else
        logger.warn "User #{current_user.uuid} tried to change owner_uuid of #{self.class.to_s} #{self.uuid} to #{self.owner_uuid} but does not have permission to write to #{self.owner_uuid}"
        return false
      end
    end
    if current_user.uuid == self.owner_uuid_was or
        current_user.uuid == self.uuid or
        current_user.can? write: self.owner_uuid_was
      # current user is, or has :write permission on, the previous owner
      return true
    else
      logger.warn "User #{current_user.uuid} tried to modify #{self.class.to_s} #{self.uuid} but does not have permission to write #{self.owner_uuid_was}"
      return false
    end
  end

  def ensure_permission_to_destroy
    raise PermissionDeniedError unless permission_to_destroy
  end

  def permission_to_destroy
    permission_to_update
  end

  def maybe_update_modified_by_fields
    update_modified_by_fields if self.changed?
  end

  def update_modified_by_fields
    self.updated_at = Time.now
    self.owner_uuid ||= current_default_owner if self.respond_to? :owner_uuid=
    self.modified_at = Time.now
    self.modified_by_user_uuid = current_user ? current_user.uuid : nil
    self.modified_by_client_uuid = current_api_client ? current_api_client.uuid : nil
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

  def foreign_key_attributes
    attributes.keys.select { |a| a.match /_uuid$/ }
  end

  def normalize_collection_uuids
    foreign_key_attributes.each do |attr|
      attr_value = send attr
      if attr_value.is_a? String and
          attr_value.match /^[0-9a-f]{32,}(\+[@\w]+)*$/
        begin
          send "#{attr}=", Collection.normalize_uuid(attr_value)
        rescue
          # TODO: abort instead of silently accepting unnormalizable value?
        end
      end
    end
  end

  def self.resource_class_for_uuid(uuid)
    if uuid.is_a? ArvadosModel
      return uuid.class
    end
    unless uuid.is_a? String
      return nil
    end
    if uuid.match /^[0-9a-f]{32}(\+[^,]+)*(,[0-9a-f]{32}(\+[^,]+)*)*$/
      return Collection
    end
    resource_class = nil

    Rails.application.eager_load!
    uuid.match /^[0-9a-z]{5}-([0-9a-z]{5})-[0-9a-z]{15}$/ do |re|
      ActiveRecord::Base.descendants.reject(&:abstract_class?).each do |k|
        if k.respond_to?(:uuid_prefix)
          if k.uuid_prefix == re[1]
            return k
          end
        end
      end
    end
    nil
  end

  def log_start_state
    @old_etag = etag
    @old_attributes = logged_attributes
  end

  def log_change(event_type)
    log = Log.new(event_type: event_type).fill_object(self)
    yield log
    log.save!
    log_start_state
  end

  def log_create
    log_change('create') do |log|
      log.fill_properties('old', nil, nil)
      log.update_to self
    end
  end

  def log_update
    log_change('update') do |log|
      log.fill_properties('old', @old_etag, @old_attributes)
      log.update_to self
    end
  end

  def log_destroy
    log_change('destroy') do |log|
      log.fill_properties('old', @old_etag, @old_attributes)
      log.update_to nil
    end
  end
end
