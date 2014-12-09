require 'has_uuid'

class ArvadosModel < ActiveRecord::Base
  self.abstract_class = true

  include CurrentApiClient      # current_user, current_api_client, etc.

  attr_protected :created_at
  attr_protected :modified_by_user_uuid
  attr_protected :modified_by_client_uuid
  attr_protected :modified_at
  after_initialize :log_start_state
  before_save :ensure_permission_to_save
  before_save :ensure_owner_uuid_is_permitted
  before_save :ensure_ownership_path_leads_to_user
  before_destroy :ensure_owner_uuid_is_permitted
  before_destroy :ensure_permission_to_destroy
  before_create :update_modified_by_fields
  before_update :maybe_update_modified_by_fields
  after_create :log_create
  after_update :log_update
  after_destroy :log_destroy
  after_find :convert_serialized_symbols_to_strings
  before_validation :normalize_collection_uuids
  validate :ensure_serialized_attribute_type
  validate :ensure_valid_uuids

  # Note: This only returns permission links. It does not account for
  # permissions obtained via user.is_admin or
  # user.uuid==object.owner_uuid.
  has_many :permissions, :foreign_key => :head_uuid, :class_name => 'Link', :primary_key => :uuid, :conditions => "link_class = 'permission'"

  class PermissionDeniedError < StandardError
    def http_status
      403
    end
  end

  class AlreadyLockedError < StandardError
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
    kind.match(/^arvados\#(.+)$/)[1].classify.safe_constantize rescue nil
  end

  def href
    "#{current_api_base}/#{self.class.to_s.pluralize.underscore}/#{self.uuid}"
  end

  def self.searchable_columns operator
    textonly_operator = !operator.match(/[<=>]/)
    self.columns.select do |col|
      case col.type
      when :string
        true
      when :text
        if operator == 'ilike'
          false
        else
          true
        end
      when :datetime, :integer, :boolean
        !textonly_operator
      else
        false
      end
    end.map(&:name)
  end

  def self.attribute_column attr
    self.columns.select { |col| col.name == attr.to_s }.first
  end

  def self.attributes_required_columns
    # This method returns a hash.  Each key is the name of an API attribute,
    # and it's mapped to a list of database columns that must be fetched
    # to generate that attribute.
    # This implementation generates a simple map of attributes to
    # matching column names.  Subclasses can override this method
    # to specify that method-backed API attributes need to fetch
    # specific columns from the database.
    all_columns = columns.map(&:name)
    api_column_map = Hash.new { |hash, key| hash[key] = [] }
    methods.grep(/^api_accessible_\w+$/).each do |method_name|
      next if method_name == :api_accessible_attributes
      send(method_name).each_pair do |api_attr_name, col_name|
        col_name = col_name.to_s
        if all_columns.include?(col_name)
          api_column_map[api_attr_name.to_s] |= [col_name]
        end
      end
    end
    api_column_map
  end

  # If current user can manage the object, return an array of uuids of
  # users and groups that have permission to write the object. The
  # first two elements are always [self.owner_uuid, current user's
  # uuid].
  #
  # If current user can write but not manage the object, return
  # [self.owner_uuid, current user's uuid].
  #
  # If current user cannot write this object, just return
  # [self.owner_uuid].
  def writable_by
    unless (owner_uuid == current_user.uuid or
            current_user.is_admin or
            (current_user.groups_i_can(:manage) & [uuid, owner_uuid]).any?)
      if ((current_user.groups_i_can(:write) + [current_user.uuid]) &
          [uuid, owner_uuid]).any?
        return [owner_uuid, current_user.uuid]
      else
        return [owner_uuid]
      end
    end
    [owner_uuid, current_user.uuid] + permissions.collect do |p|
      if ['can_write', 'can_manage'].index p.name
        p.tail_uuid
      end
    end.compact.uniq
  end

  # Return a query with read permissions restricted to the union of of the
  # permissions of the members of users_list, i.e. if something is readable by
  # any user in users_list, it will be readable in the query returned by this
  # function.
  def self.readable_by(*users_list)
    # Get rid of troublesome nils
    users_list.compact!

    # Load optional keyword arguments, if they exist.
    if users_list.last.is_a? Hash
      kwargs = users_list.pop
    else
      kwargs = {}
    end

    # Check if any of the users are admin.  If so, we're done.
    if users_list.select { |u| u.is_admin }.any?
      return self
    end

    # Collect the uuids for each user and any groups readable by each user.
    user_uuids = users_list.map { |u| u.uuid }
    uuid_list = user_uuids + users_list.flat_map { |u| u.groups_i_can(:read) }
    sql_conds = []
    sql_params = []
    sql_table = kwargs.fetch(:table_name, table_name)
    or_object_uuid = ''

    # This row is owned by a member of users_list, or owned by a group
    # readable by a member of users_list
    # or
    # This row uuid is the uuid of a member of users_list
    # or
    # A permission link exists ('write' and 'manage' implicitly include
    # 'read') from a member of users_list, or a group readable by users_list,
    # to this row, or to the owner of this row (see join() below).
    sql_conds += ["#{sql_table}.uuid in (?)"]
    sql_params += [user_uuids]

    if uuid_list.any?
      sql_conds += ["#{sql_table}.owner_uuid in (?)"]
      sql_params += [uuid_list]

      sanitized_uuid_list = uuid_list.
        collect { |uuid| sanitize(uuid) }.join(', ')
      permitted_uuids = "(SELECT head_uuid FROM links WHERE link_class='permission' AND tail_uuid IN (#{sanitized_uuid_list}))"
      sql_conds += ["#{sql_table}.uuid IN #{permitted_uuids}"]
    end

    if sql_table == "links" and users_list.any?
      # This row is a 'permission' or 'resources' link class
      # The uuid for a member of users_list is referenced in either the head
      # or tail of the link
      sql_conds += ["(#{sql_table}.link_class in (#{sanitize 'permission'}, #{sanitize 'resources'}) AND (#{sql_table}.head_uuid IN (?) OR #{sql_table}.tail_uuid IN (?)))"]
      sql_params += [user_uuids, user_uuids]
    end

    if sql_table == "logs" and users_list.any?
      # Link head points to the object described by this row
      sql_conds += ["#{sql_table}.object_uuid IN #{permitted_uuids}"]

      # This object described by this row is owned by this user, or owned by a group readable by this user
      sql_conds += ["#{sql_table}.object_owner_uuid in (?)"]
      sql_params += [uuid_list]
    end

    # Link head points to this row, or to the owner of this row (the
    # thing to be read)
    #
    # Link tail originates from this user, or a group that is readable
    # by this user (the identity with authorization to read)
    #
    # Link class is 'permission' ('write' and 'manage' implicitly
    # include 'read')
    where(sql_conds.join(' OR '), *sql_params)
  end

  def logged_attributes
    attributes
  end

  protected

  def ensure_ownership_path_leads_to_user
    if new_record? or owner_uuid_changed?
      uuid_in_path = {owner_uuid => true, uuid => true}
      x = owner_uuid
      while (owner_class = ArvadosModel::resource_class_for_uuid(x)) != User
        begin
          if x == uuid
            # Test for cycles with the new version, not the DB contents
            x = owner_uuid
          elsif !owner_class.respond_to? :find_by_uuid
            raise ActiveRecord::RecordNotFound.new
          else
            x = owner_class.find_by_uuid(x).owner_uuid
          end
        rescue ActiveRecord::RecordNotFound => e
          errors.add :owner_uuid, "is not owned by any user: #{e}"
          return false
        end
        if uuid_in_path[x]
          if x == owner_uuid
            errors.add :owner_uuid, "would create an ownership cycle"
          else
            errors.add :owner_uuid, "has an ownership cycle"
          end
          return false
        end
        uuid_in_path[x] = true
      end
    end
    true
  end

  def ensure_owner_uuid_is_permitted
    raise PermissionDeniedError if !current_user

    if new_record? and respond_to? :owner_uuid=
      self.owner_uuid ||= current_user.uuid
    end

    if self.owner_uuid.nil?
      errors.add :owner_uuid, "cannot be nil"
      raise PermissionDeniedError
    end

    rsc_class = ArvadosModel::resource_class_for_uuid owner_uuid
    unless rsc_class == User or rsc_class == Group
      errors.add :owner_uuid, "must be set to User or Group"
      raise PermissionDeniedError
    end

    # Verify "write" permission on old owner
    # default fail unless one of:
    # owner_uuid did not change
    # previous owner_uuid is nil
    # current user is the old owner
    # current user is this object
    # current user can_write old owner
    unless !owner_uuid_changed? or
        owner_uuid_was.nil? or
        current_user.uuid == self.owner_uuid_was or
        current_user.uuid == self.uuid or
        current_user.can? write: self.owner_uuid_was
      logger.warn "User #{current_user.uuid} tried to modify #{self.class.to_s} #{uuid} but does not have permission to write old owner_uuid #{owner_uuid_was}"
      errors.add :owner_uuid, "cannot be changed without write permission on old owner"
      raise PermissionDeniedError
    end

    # Verify "write" permission on new owner
    # default fail unless one of:
    # current_user is this object
    # current user can_write new owner
    unless current_user == self or current_user.can? write: owner_uuid
      logger.warn "User #{current_user.uuid} tried to modify #{self.class.to_s} #{uuid} but does not have permission to write new owner_uuid #{owner_uuid}"
      errors.add :owner_uuid, "cannot be changed without write permission on new owner"
      raise PermissionDeniedError
    end

    true
  end

  def ensure_permission_to_save
    unless (new_record? ? permission_to_create : permission_to_update)
      raise PermissionDeniedError
    end
  end

  def permission_to_create
    current_user.andand.is_active
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
    return true
  end

  def ensure_permission_to_destroy
    raise PermissionDeniedError unless permission_to_destroy
  end

  def permission_to_destroy
    permission_to_update
  end

  def maybe_update_modified_by_fields
    update_modified_by_fields if self.changed? or self.new_record?
    true
  end

  def update_modified_by_fields
    self.updated_at = Time.now
    self.owner_uuid ||= current_default_owner if self.respond_to? :owner_uuid=
    self.modified_at = Time.now
    self.modified_by_user_uuid = current_user ? current_user.uuid : nil
    self.modified_by_client_uuid = current_api_client ? current_api_client.uuid : nil
    true
  end

  def self.has_symbols? x
    if x.is_a? Hash
      x.each do |k,v|
        return true if has_symbols?(k) or has_symbols?(v)
      end
      false
    elsif x.is_a? Array
      x.each do |k|
        return true if has_symbols?(k)
      end
      false
    else
      (x.class == Symbol)
    end
  end

  def self.recursive_stringify x
    if x.is_a? Hash
      Hash[x.collect do |k,v|
             [recursive_stringify(k), recursive_stringify(v)]
           end]
    elsif x.is_a? Array
      x.collect do |k|
        recursive_stringify k
      end
    elsif x.is_a? Symbol
      x.to_s
    else
      x
    end
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
        if self.attributes[colname].class != attr.object_class
          self.errors.add colname.to_sym, "must be a #{attr.object_class.to_s}, not a #{self.attributes[colname].class.to_s}"
        elsif self.class.has_symbols? attributes[colname]
          self.errors.add colname.to_sym, "must not contain symbols: #{attributes[colname].inspect}"
        end
      end
    end
  end

  def convert_serialized_symbols_to_strings
    # ensure_serialized_attribute_type should prevent symbols from
    # getting into the database in the first place. If someone managed
    # to get them into the database (perhaps using an older version)
    # we'll convert symbols to strings when loading from the
    # database. (Otherwise, loading and saving an object with existing
    # symbols in a serialized field will crash.)
    self.class.serialized_attributes.each do |colname, attr|
      if self.class.has_symbols? attributes[colname]
        attributes[colname] = self.class.recursive_stringify attributes[colname]
        self.send(colname + '=',
                  self.class.recursive_stringify(attributes[colname]))
      end
    end
  end

  def foreign_key_attributes
    attributes.keys.select { |a| a.match /_uuid$/ }
  end

  def skip_uuid_read_permission_check
    %w(modified_by_client_uuid)
  end

  def skip_uuid_existence_check
    []
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

  @@prefixes_hash = nil
  def self.uuid_prefixes
    unless @@prefixes_hash
      @@prefixes_hash = {}
      ActiveRecord::Base.descendants.reject(&:abstract_class?).each do |k|
        if k.respond_to?(:uuid_prefix)
          @@prefixes_hash[k.uuid_prefix] = k
        end
      end
    end
    @@prefixes_hash
  end

  def self.uuid_like_pattern
    "_____-#{uuid_prefix}-_______________"
  end

  def ensure_valid_uuids
    specials = [system_user_uuid]

    foreign_key_attributes.each do |attr|
      if new_record? or send (attr + "_changed?")
        next if skip_uuid_existence_check.include? attr
        attr_value = send attr
        next if specials.include? attr_value
        if attr_value
          if (r = ArvadosModel::resource_class_for_uuid attr_value)
            unless skip_uuid_read_permission_check.include? attr
              r = r.readable_by(current_user)
            end
            if r.where(uuid: attr_value).count == 0
              errors.add(attr, "'#{attr_value}' not found")
            end
          end
        end
      end
    end
  end

  class Email
    def self.kind
      "email"
    end

    def kind
      self.class.kind
    end

    def self.readable_by (*u)
      self
    end

    def self.where (u)
      [{:uuid => u[:uuid]}]
    end
  end

  def self.resource_class_for_uuid(uuid)
    if uuid.is_a? ArvadosModel
      return uuid.class
    end
    unless uuid.is_a? String
      return nil
    end
    resource_class = nil

    Rails.application.eager_load!
    uuid.match HasUuid::UUID_REGEX do |re|
      return uuid_prefixes[re[1]] if uuid_prefixes[re[1]]
    end

    if uuid.match /.+@.+/
      return Email
    end

    nil
  end

  # ArvadosModel.find_by_uuid needs extra magic to allow it to return
  # an object in any class.
  def self.find_by_uuid uuid
    if self == ArvadosModel
      # If called directly as ArvadosModel.find_by_uuid rather than via subclass,
      # delegate to the appropriate subclass based on the given uuid.
      self.resource_class_for_uuid(uuid).find_by_uuid(uuid)
    else
      super
    end
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
