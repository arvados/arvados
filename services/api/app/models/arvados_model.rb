# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

require 'arvados_model_updates'
require 'has_uuid'
require 'record_filters'
require 'serializers'
require 'request_error'

class ArvadosModel < ActiveRecord::Base
  self.abstract_class = true

  include ArvadosModelUpdates
  include CurrentApiClient      # current_user, current_api_client, etc.
  include DbCurrentTime
  extend RecordFilters

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
  before_validation :set_default_owner
  validate :ensure_valid_uuids

  # Note: This only returns permission links. It does not account for
  # permissions obtained via user.is_admin or
  # user.uuid==object.owner_uuid.
  has_many(:permissions,
           ->{where(link_class: 'permission')},
           foreign_key: :head_uuid,
           class_name: 'Link',
           primary_key: :uuid)

  class PermissionDeniedError < RequestError
    def http_status
      403
    end
  end

  class AlreadyLockedError < RequestError
    def http_status
      422
    end
  end

  class LockFailedError < RequestError
    def http_status
      422
    end
  end

  class InvalidStateTransitionError < RequestError
    def http_status
      422
    end
  end

  class UnauthorizedError < RequestError
    def http_status
      401
    end
  end

  class UnresolvableContainerError < RequestError
    def http_status
      422
    end
  end

  def self.kind_class(kind)
    kind.match(/^arvados\#(.+)$/)[1].classify.safe_constantize rescue nil
  end

  def href
    "#{current_api_base}/#{self.class.to_s.pluralize.underscore}/#{self.uuid}"
  end

  def self.permit_attribute_params raw_params
    # strong_parameters does not provide security: permissions are
    # implemented with before_save hooks.
    #
    # The following permit! is necessary even with
    # "ActionController::Parameters.permit_all_parameters = true",
    # because permit_all does not permit nested attributes.
    if raw_params
      serialized_attributes.each do |colname, coder|
        param = raw_params[colname.to_sym]
        if param.nil?
          # ok
        elsif !param.is_a?(coder.object_class)
          raise ArgumentError.new("#{colname} parameter must be #{coder.object_class}, not #{param.class}")
        elsif has_nonstring_keys?(param)
          raise ArgumentError.new("#{colname} parameter cannot have non-string hash keys")
        end
      end
    end
    ActionController::Parameters.new(raw_params).permit!
  end

  def initialize raw_params={}, *args
    super(self.class.permit_attribute_params(raw_params), *args)
  end

  # Reload "old attributes" for logging, too.
  def reload(*args)
    super
    log_start_state
  end

  def self.create raw_params={}, *args
    super(permit_attribute_params(raw_params), *args)
  end

  def update_attributes raw_params={}, *args
    super(self.class.permit_attribute_params(raw_params), *args)
  end

  def self.selectable_attributes(template=:user)
    # Return an array of attribute name strings that can be selected
    # in the given template.
    api_accessible_attributes(template).map { |attr_spec| attr_spec.first.to_s }
  end

  def self.searchable_columns operator
    textonly_operator = !operator.match(/[<=>]/)
    self.columns.select do |col|
      case col.type
      when :string, :text
        true
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

  def self.ignored_select_attributes
    ["href", "kind", "etag"]
  end

  def self.columns_for_attributes(select_attributes)
    if select_attributes.empty?
      raise ArgumentError.new("Attribute selection list cannot be empty")
    end
    api_column_map = attributes_required_columns
    invalid_attrs = []
    select_attributes.each do |s|
      next if ignored_select_attributes.include? s
      if not s.is_a? String or not api_column_map.include? s
        invalid_attrs << s
      end
    end
    if not invalid_attrs.empty?
      raise ArgumentError.new("Invalid attribute(s): #{invalid_attrs.inspect}")
    end
    # Given an array of attribute names to select, return an array of column
    # names that must be fetched from the database to satisfy the request.
    select_attributes.flat_map { |attr| api_column_map[attr] }.uniq
  end

  def self.default_orders
    ["#{table_name}.modified_at desc", "#{table_name}.uuid"]
  end

  def self.unique_columns
    ["id", "uuid"]
  end

  def self.limit_index_columns_read
    # This method returns a list of column names.
    # If an index request reads that column from the database,
    # APIs that return lists will only fetch objects until reaching
    # max_index_database_read bytes of data from those columns.
    []
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
    return [owner_uuid] if not current_user
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

    # Collect the UUIDs of the authorized users.
    sql_table = kwargs.fetch(:table_name, table_name)
    include_trash = kwargs.fetch(:include_trash, false)

    sql_conds = nil
    user_uuids = users_list.map { |u| u.uuid }

    exclude_trashed_records = ""
    if !include_trash and (sql_table == "groups" or sql_table == "collections") then
      # Only include records that are not explicitly trashed
      exclude_trashed_records = "AND #{sql_table}.is_trashed = false"
    end

    if users_list.select { |u| u.is_admin }.any?
      # Admin skips most permission checks, but still want to filter on trashed items.
      if !include_trash
        if sql_table != "api_client_authorizations"
          # Only include records where the owner is not trashed
          sql_conds = "NOT EXISTS(SELECT 1 FROM #{PERMISSION_VIEW} "+
                      "WHERE trashed = 1 AND "+
                      "(#{sql_table}.owner_uuid = target_uuid)) #{exclude_trashed_records}"
        end
      end
    else
      trashed_check = ""
      if !include_trash then
        trashed_check = "AND trashed = 0"
      end

      # Note: it is possible to combine the direct_check and
      # owner_check into a single EXISTS() clause, however it turns
      # out query optimizer doesn't like it and forces a sequential
      # table scan.  Constructing the query with separate EXISTS()
      # clauses enables it to use the index.
      #
      # see issue 13208 for details.

      # Match a direct read permission link from the user to the record uuid
      direct_check = "EXISTS(SELECT 1 FROM #{PERMISSION_VIEW} "+
                     "WHERE user_uuid IN (:user_uuids) AND perm_level >= 1 #{trashed_check} AND target_uuid = #{sql_table}.uuid)"

      # Match a read permission link from the user to the record's owner_uuid
      owner_check = ""
      if sql_table != "api_client_authorizations" and sql_table != "groups" then
        owner_check = "OR EXISTS(SELECT 1 FROM #{PERMISSION_VIEW} "+
          "WHERE user_uuid IN (:user_uuids) AND perm_level >= 1 #{trashed_check} AND target_uuid = #{sql_table}.owner_uuid AND target_owner_uuid IS NOT NULL) "
      end

      links_cond = ""
      if sql_table == "links"
        # Match any permission link that gives one of the authorized
        # users some permission _or_ gives anyone else permission to
        # view one of the authorized users.
        links_cond = "OR (#{sql_table}.link_class IN (:permission_link_classes) AND "+
                       "(#{sql_table}.head_uuid IN (:user_uuids) OR #{sql_table}.tail_uuid IN (:user_uuids)))"
      end

      sql_conds = "(#{direct_check} #{owner_check} #{links_cond}) #{exclude_trashed_records}"

    end

    self.where(sql_conds,
               user_uuids: user_uuids,
               permission_link_classes: ['permission', 'resources'])
  end

  def save_with_unique_name!
    uuid_was = uuid
    name_was = name
    max_retries = 2
    transaction do
      conn = ActiveRecord::Base.connection
      conn.exec_query 'SAVEPOINT save_with_unique_name'
      begin
        save!
      rescue ActiveRecord::RecordNotUnique => rn
        raise if max_retries == 0
        max_retries -= 1

        conn.exec_query 'ROLLBACK TO SAVEPOINT save_with_unique_name'

        # Dig into the error to determine if it is specifically calling out a
        # (owner_uuid, name) uniqueness violation.  In this specific case, and
        # the client requested a unique name with ensure_unique_name==true,
        # update the name field and try to save again.  Loop as necessary to
        # discover a unique name.  It is necessary to handle name choosing at
        # this level (as opposed to the client) to ensure that record creation
        # never fails due to a race condition.
        err = rn.original_exception
        raise unless err.is_a?(PG::UniqueViolation)

        # Unfortunately ActiveRecord doesn't abstract out any of the
        # necessary information to figure out if this the error is actually
        # the specific case where we want to apply the ensure_unique_name
        # behavior, so the following code is specialized to Postgres.
        detail = err.result.error_field(PG::Result::PG_DIAG_MESSAGE_DETAIL)
        raise unless /^Key \(owner_uuid, name\)=\([a-z0-9]{5}-[a-z0-9]{5}-[a-z0-9]{15}, .*?\) already exists\./.match detail

        new_name = "#{name_was} (#{db_current_time.utc.iso8601(3)})"
        if new_name == name
          # If the database is fast enough to do two attempts in the
          # same millisecond, we need to wait to ensure we try a
          # different timestamp on each attempt.
          sleep 0.002
          new_name = "#{name_was} (#{db_current_time.utc.iso8601(3)})"
        end

        self[:name] = new_name
        self[:uuid] = nil if uuid_was.nil? && !uuid.nil?
        conn.exec_query 'SAVEPOINT save_with_unique_name'
        retry
      ensure
        conn.exec_query 'RELEASE SAVEPOINT save_with_unique_name'
      end
    end
  end

  def logged_attributes
    attributes.except(*Rails.configuration.unlogged_attributes)
  end

  def self.full_text_searchable_columns
    self.columns.select do |col|
      [:string, :text, :jsonb].include?(col.type)
    end.map(&:name)
  end

  def self.full_text_tsvector
    parts = full_text_searchable_columns.collect do |column|
      cast = serialized_attributes[column] ? '::text' : ''
      "coalesce(#{column}#{cast},'')"
    end
    "to_tsvector('english', #{parts.join(" || ' ' || ")})"
  end

  def self.apply_filters query, filters
    ft = record_filters filters, self
    if not ft[:cond_out].any?
      return query
    end
    query.where('(' + ft[:cond_out].join(') AND (') + ')',
                          *ft[:param_out])
  end

  protected

  def self.deep_sort_hash(x)
    if x.is_a? Hash
      x.sort.collect do |k, v|
        [k, deep_sort_hash(v)]
      end.to_h
    elsif x.is_a? Array
      x.collect { |v| deep_sort_hash(v) }
    else
      x
    end
  end

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

  def set_default_owner
    if new_record? and current_user and respond_to? :owner_uuid=
      self.owner_uuid ||= current_user.uuid
    end
  end

  def ensure_owner_uuid_is_permitted
    raise PermissionDeniedError if !current_user

    if self.owner_uuid.nil?
      errors.add :owner_uuid, "cannot be nil"
      raise PermissionDeniedError
    end

    rsc_class = ArvadosModel::resource_class_for_uuid owner_uuid
    unless rsc_class == User or rsc_class == Group
      errors.add :owner_uuid, "must be set to User or Group"
      raise PermissionDeniedError
    end

    if new_record? || owner_uuid_changed?
      # Permission on owner_uuid_was is needed to move an existing
      # object away from its previous owner (which implies permission
      # to modify this object itself, so we don't need to check that
      # separately). Permission on the new owner_uuid is also needed.
      [['old', owner_uuid_was],
       ['new', owner_uuid]
      ].each do |which, check_uuid|
        if check_uuid.nil?
          # old_owner_uuid is nil? New record, no need to check.
        elsif !current_user.can?(write: check_uuid)
          logger.warn "User #{current_user.uuid} tried to set ownership of #{self.class.to_s} #{self.uuid} but does not have permission to write #{which} owner_uuid #{check_uuid}"
          errors.add :owner_uuid, "cannot be set or changed without write permission on #{which} owner"
          raise PermissionDeniedError
        end
      end
    else
      # If the object already existed and we're not changing
      # owner_uuid, we only need write permission on the object
      # itself.
      if !current_user.can?(write: self.uuid)
        logger.warn "User #{current_user.uuid} tried to modify #{self.class.to_s} #{self.uuid} without write permission"
        errors.add :uuid, "is not writable"
        raise PermissionDeniedError
      end
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
    current_time = db_current_time
    self.created_at = created_at_was || current_time
    self.updated_at = current_time
    self.owner_uuid ||= current_default_owner if self.respond_to? :owner_uuid=
    self.modified_at = current_time
    if !anonymous_updater
      self.modified_by_user_uuid = current_user ? current_user.uuid : nil
    end
    self.modified_by_client_uuid = current_api_client ? current_api_client.uuid : nil
    true
  end

  def self.has_nonstring_keys? x
    if x.is_a? Hash
      x.each do |k,v|
        return true if !(k.is_a?(String) || k.is_a?(Symbol)) || has_nonstring_keys?(v)
      end
    elsif x.is_a? Array
      x.each do |v|
        return true if has_nonstring_keys?(v)
      end
    end
    false
  end

  def self.has_symbols? x
    if x.is_a? Hash
      x.each do |k,v|
        return true if has_symbols?(k) or has_symbols?(v)
      end
    elsif x.is_a? Array
      x.each do |k|
        return true if has_symbols?(k)
      end
    elsif x.is_a? Symbol
      return true
    elsif x.is_a? String
      return true if x.start_with?(':') && !x.start_with?('::')
    end
    false
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
    elsif x.is_a? String and x.start_with?(':') and !x.start_with?('::')
      x[1..-1]
    else
      x
    end
  end

  def self.where_serialized(colname, value)
    if value.empty?
      # rails4 stores as null, rails3 stored as serialized [] or {}
      sql = "#{colname.to_s} is null or #{colname.to_s} IN (?)"
      sorted = value
    else
      sql = "#{colname.to_s} IN (?)"
      sorted = deep_sort_hash(value)
    end
    where(sql, [sorted.to_yaml, SafeJSON.dump(sorted)])
  end

  Serializer = {
    Hash => HashSerializer,
    Array => ArraySerializer,
  }

  def self.serialize(colname, type)
    coder = Serializer[type]
    @serialized_attributes ||= {}
    @serialized_attributes[colname.to_s] = coder
    super(colname, coder)
  end

  def self.serialized_attributes
    @serialized_attributes ||= {}
  end

  def serialized_attributes
    self.class.serialized_attributes
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
        send(colname + '=',
             self.class.recursive_stringify(attributes[colname]))
      end
    end
  end

  def foreign_key_attributes
    attributes.keys.select { |a| a.match(/_uuid$/) }
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
          attr_value.match(/^[0-9a-f]{32,}(\+[@\w]+)*$/)
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
      Rails.application.eager_load!
      ActiveRecord::Base.descendants.reject(&:abstract_class?).each do |k|
        if k.respond_to?(:uuid_prefix)
          @@prefixes_hash[k.uuid_prefix] = k
        end
      end
    end
    @@prefixes_hash
  end

  def self.uuid_like_pattern
    "#{Rails.configuration.uuid_prefix}-#{uuid_prefix}-_______________"
  end

  def self.uuid_regex
    %r/[a-z0-9]{5}-#{uuid_prefix}-[a-z0-9]{15}/
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

    uuid.match HasUuid::UUID_REGEX do |re|
      return uuid_prefixes[re[1]] if uuid_prefixes[re[1]]
    end

    if uuid.match(/.+@.+/)
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
    @old_attributes = Marshal.load(Marshal.dump(attributes))
    @old_logged_attributes = Marshal.load(Marshal.dump(logged_attributes))
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
      log.fill_properties('old', etag(@old_attributes), @old_logged_attributes)
      log.update_to self
    end
  end

  def log_destroy
    log_change('delete') do |log|
      log.fill_properties('old', etag(@old_attributes), @old_logged_attributes)
      log.update_to nil
    end
  end
end
