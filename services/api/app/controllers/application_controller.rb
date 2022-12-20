# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

require 'safe_json'
require 'request_error'

module ApiTemplateOverride
  def allowed_to_render?(fieldset, field, model, options)
    return false if !super
    if options[:select]
      options[:select].include? field.to_s
    else
      true
    end
  end
end

class ActsAsApi::ApiTemplate
  prepend ApiTemplateOverride
end

require 'load_param'

class ApplicationController < ActionController::Base
  include ThemesForRails::ActionController
  include CurrentApiClient
  include LoadParam
  include DbCurrentTime

  respond_to :json
  protect_from_forgery

  ERROR_ACTIONS = [:render_error, :render_not_found]

  around_action :set_current_request_id
  before_action :disable_api_methods
  before_action :set_cors_headers
  before_action :respond_with_json_by_default
  before_action :remote_ip
  before_action :load_read_auths
  before_action :require_auth_scope, except: ERROR_ACTIONS

  before_action :catch_redirect_hint
  before_action :load_required_parameters
  before_action :load_limit_offset_order_params, only: [:index, :contents]
  before_action :load_select_param
  before_action(:find_object_by_uuid,
                except: [:index, :create, :update] + ERROR_ACTIONS)
  before_action :find_object_for_update, only: [:update]
  before_action :load_where_param, only: [:index, :contents]
  before_action :load_filters_param, only: [:index, :contents]
  before_action :find_objects_for_index, :only => :index
  before_action(:set_nullable_attrs_to_null, only: [:update, :create])
  before_action :reload_object_before_update, :only => :update
  before_action(:render_404_if_no_object,
                except: [:index, :create] + ERROR_ACTIONS)
  before_action :only_admin_can_bypass_federation

  attr_writer :resource_attrs

  begin
    rescue_from(Exception,
                ArvadosModel::PermissionDeniedError,
                :with => :render_error)
    rescue_from(ActiveRecord::RecordNotFound,
                ActionController::RoutingError,
                AbstractController::ActionNotFound,
                :with => :render_not_found)
  end

  def initialize *args
    super
    @object = nil
    @objects = nil
    @offset = nil
    @limit = nil
    @select = nil
    @distinct = nil
    @response_resource_name = nil
    @attrs = nil
    @extra_included = nil
  end

  def default_url_options
    options = {}
    if Rails.configuration.Services.Controller.ExternalURL != URI("")
      exturl = Rails.configuration.Services.Controller.ExternalURL
      options[:host] = exturl.host
      options[:port] = exturl.port
      options[:protocol] = exturl.scheme
    end
    options
  end

  def index
    if params[:eager] and params[:eager] != '0' and params[:eager] != 0 and params[:eager] != ''
      @objects.each(&:eager_load_associations)
    end
    render_list
  end

  def show
    send_json @object.as_api_response(nil, select: @select)
  end

  def create
    @object = model_class.new resource_attrs

    if @object.respond_to?(:name) && params[:ensure_unique_name]
      @object.save_with_unique_name!
    else
      @object.save!
    end

    show
  end

  def update
    attrs_to_update = resource_attrs.reject { |k,v|
      [:kind, :etag, :href].index k
    }
    @object.update_attributes! attrs_to_update
    show
  end

  def destroy
    @object.destroy
    show
  end

  def catch_redirect_hint
    if !current_user
      if params.has_key?('redirect_to') then
        session[:redirect_to] = params[:redirect_to]
      end
    end
  end

  def render_404_if_no_object
    render_not_found "Object not found" if !@object
  end

  def only_admin_can_bypass_federation
    unless !params[:bypass_federation] || current_user.andand.is_admin
      send_error("The bypass_federation parameter is only permitted when current user is admin", status: 403)
    end
  end

  def render_error(e)
    logger.error e.inspect
    if e.respond_to? :backtrace and e.backtrace
      # This will be cleared by lograge after adding it to the log.
      # Usually lograge would get the exceptions, but in our case we're catching
      # all of them with exception handlers that cannot re-raise them because they
      # don't get propagated.
      Thread.current[:exception] = e.inspect
      Thread.current[:backtrace] = e.backtrace.collect { |x| x + "\n" }.join('')
    end
    if (@object.respond_to? :errors and
        @object.errors.andand.full_messages.andand.any?)
      errors = @object.errors.full_messages
      logger.error errors.inspect
    else
      errors = [e.inspect]
    end
    status = e.respond_to?(:http_status) ? e.http_status : 422
    send_error(*errors, status: status)
  end

  def render_not_found(e=ActionController::RoutingError.new("Path not found"))
    logger.error e.inspect
    send_error("Path not found", status: 404)
  end

  def render_accepted
    send_json ({accepted: true}), status: 202
  end

  protected

  def bool_param(pname)
    if params.include?(pname)
      if params[pname].is_a?(Boolean)
        return params[pname]
      else
        logger.warn "Warning: received non-boolean value #{params[pname].inspect} for boolean parameter #{pname} on #{self.class.inspect}, treating as false."
      end
    end
    false
  end

  def send_error(*args)
    if args.last.is_a? Hash
      err = args.pop
    else
      err = {}
    end
    err[:errors] ||= args
    err[:errors].map! do |err|
      err += " (#{request.request_id})"
    end
    err[:error_token] = [Time.now.utc.to_i, "%08x" % rand(16 ** 8)].join("+")
    status = err.delete(:status) || 422
    logger.error "Error #{err[:error_token]}: #{status}"
    send_json err, status: status
  end

  def send_json response, opts={}
    # The obvious render(json: ...) forces a slow JSON encoder. See
    # #3021 and commit logs. Might be fixed in Rails 4.1.
    render({
             plain: SafeJSON.dump(response).html_safe,
             content_type: 'application/json'
           }.merge opts)
  end

  def find_objects_for_index
    @objects ||= model_class.readable_by(*@read_users, {
      :include_trash => (bool_param(:include_trash) || 'untrash' == action_name),
      :include_old_versions => bool_param(:include_old_versions)
    })
    apply_where_limit_order_params
  end

  def apply_filters model_class=nil
    model_class ||= self.model_class
    @objects = model_class.apply_filters(@objects, @filters)
  end

  def apply_where_limit_order_params model_class=nil
    model_class ||= self.model_class
    apply_filters model_class

    ar_table_name = @objects.table_name
    if @where.is_a? Hash and @where.any?
      conditions = ['1=1']
      @where.each do |attr,value|
        if attr.to_s == 'any'
          if value.is_a?(Array) and
              value.length == 2 and
              value[0] == 'contains' then
            ilikes = []
            model_class.searchable_columns('ilike').each do |column|
              # Including owner_uuid in an "any column" search will
              # probably just return a lot of false positives.
              next if column == 'owner_uuid'
              ilikes << "#{ar_table_name}.#{column} ilike ?"
              conditions << "%#{value[1]}%"
            end
            if ilikes.any?
              conditions[0] << ' and (' + ilikes.join(' or ') + ')'
            end
          end
        elsif attr.to_s.match(/^[a-z][_a-z0-9]+$/) and
            model_class.columns.collect(&:name).index(attr.to_s)
          if value.nil?
            conditions[0] << " and #{ar_table_name}.#{attr} is ?"
            conditions << nil
          elsif value.is_a? Array
            if value[0] == 'contains' and value.length == 2
              conditions[0] << " and #{ar_table_name}.#{attr} like ?"
              conditions << "%#{value[1]}%"
            else
              conditions[0] << " and #{ar_table_name}.#{attr} in (?)"
              conditions << value
            end
          elsif value.is_a? String or value.is_a? Integer or value == true or value == false
            conditions[0] << " and #{ar_table_name}.#{attr}=?"
            conditions << value
          elsif value.is_a? Hash
            # Not quite the same thing as "equal?" but better than nothing?
            value.each do |k,v|
              if v.is_a? String
                conditions[0] << " and #{ar_table_name}.#{attr} ilike ?"
                conditions << "%#{k}%#{v}%"
              end
            end
          end
        end
      end
      if conditions.length > 1
        conditions[0].sub!(/^1=1 and /, '')
        @objects = @objects.
          where(*conditions)
      end
    end

    if @select
      unless action_name.in? %w(create update destroy)
        # Map attribute names in @select to real column names, resolve
        # those to fully-qualified SQL column names, and pass the
        # resulting string to the select method.
        columns_list = model_class.columns_for_attributes(@select).
          map { |s| "#{ar_table_name}.#{ActiveRecord::Base.connection.quote_column_name s}" }
        @objects = @objects.select(columns_list.join(", "))
      end

      # This information helps clients understand what they're seeing
      # (Workbench always expects it), but they can't select it explicitly
      # because it's not an SQL column.  Always add it.
      # (This is harmless, given that clients can deduce what they're
      # looking at by the returned UUID anyway.)
      @select |= ["kind"]
    end
    @objects = @objects.order(@orders.join ", ") if @orders.any?
    @objects = @objects.limit(@limit)
    @objects = @objects.offset(@offset)
    @objects = @objects.distinct() if @distinct
  end

  # limit_database_read ensures @objects (which must be an
  # ActiveRelation) does not return too many results to fit in memory,
  # by previewing the results and calling @objects.limit() if
  # necessary.
  def limit_database_read(model_class:)
    return if @limit == 0 || @limit == 1
    model_class ||= self.model_class
    limit_columns = model_class.limit_index_columns_read
    limit_columns &= model_class.columns_for_attributes(@select) if @select
    return if limit_columns.empty?
    model_class.transaction do
      limit_query = @objects.
        except(:select, :distinct).
        select("(%s) as read_length" %
               limit_columns.map { |s| "octet_length(#{model_class.table_name}.#{s})" }.join(" + "))
      new_limit = 0
      read_total = 0
      limit_query.each do |record|
        new_limit += 1
        read_total += record.read_length.to_i
        if read_total >= Rails.configuration.API.MaxIndexDatabaseRead
          new_limit -= 1 if new_limit > 1
          @limit = new_limit
          break
        elsif new_limit >= @limit
          break
        end
      end
      @objects = @objects.limit(@limit)
      # Force @objects to run its query inside this transaction.
      @objects.each { |_| break }
    end
  end

  def resource_attrs
    return @attrs if @attrs
    @attrs = params[resource_name]
    if @attrs.nil?
      @attrs = {}
    elsif @attrs.is_a? String
      @attrs = Oj.strict_load @attrs, symbol_keys: true
    end
    unless [Hash, ActionController::Parameters].include? @attrs.class
      message = "No #{resource_name}"
      if resource_name.index('_')
        message << " (or #{resource_name.camelcase(:lower)})"
      end
      message << " hash provided with request"
      raise ArgumentError.new(message)
    end
    %w(created_at modified_by_client_uuid modified_by_user_uuid modified_at).each do |x|
      @attrs.delete x.to_sym
    end
    @attrs = @attrs.symbolize_keys if @attrs.is_a? ActiveSupport::HashWithIndifferentAccess
    @attrs
  end

  # Authentication
  def load_read_auths
    @read_auths = []
    if current_api_client_authorization
      @read_auths << current_api_client_authorization
    end
    # Load reader tokens if this is a read request.
    # If there are too many reader tokens, assume the request is malicious
    # and ignore it.
    if request.get? and params[:reader_tokens] and
      params[:reader_tokens].size < 100
      secrets = params[:reader_tokens].map { |t|
        if t.is_a? String and t.starts_with? "v2/"
          t.split("/")[2]
        else
          t
        end
      }
      @read_auths += ApiClientAuthorization
        .includes(:user)
        .where('api_token IN (?) AND
                (expires_at IS NULL OR expires_at > CURRENT_TIMESTAMP)',
               secrets)
        .to_a
    end
    @read_auths.select! { |auth| auth.scopes_allow_request? request }
    @read_users = @read_auths.map(&:user).uniq
  end

  def require_login
    if not current_user
      respond_to do |format|
        format.json { send_error("Not logged in", status: 401) }
        format.html { redirect_to '/login' }
      end
      false
    end
  end

  def admin_required
    unless current_user and current_user.is_admin
      send_error("Forbidden", status: 403)
    end
  end

  def require_auth_scope
    unless current_user && @read_auths.any? { |auth| auth.user.andand.uuid == current_user.uuid }
      if require_login != false
        send_error("Forbidden", status: 403)
      end
      false
    end
  end

  def set_current_request_id
    Rails.logger.tagged(request.request_id) do
      yield
    end
  end

  def append_info_to_payload(payload)
    super
    payload[:request_id] = request.request_id
    payload[:client_ipaddr] = @remote_ip
    payload[:client_auth] = current_api_client_authorization.andand.uuid || nil
  end

  def disable_api_methods
    if Rails.configuration.API.DisabledAPIs[controller_name + "." + action_name]
      send_error("Disabled", status: 404)
    end
  end

  def set_cors_headers
    response.headers['Access-Control-Allow-Origin'] = '*'
    response.headers['Access-Control-Allow-Methods'] = 'GET, HEAD, PUT, POST, DELETE'
    response.headers['Access-Control-Allow-Headers'] = 'Authorization, Content-Type'
    response.headers['Access-Control-Max-Age'] = '86486400'
  end

  def respond_with_json_by_default
    html_index = request.accepts.index(Mime[:html])
    if html_index.nil? or request.accepts[0...html_index].include?(Mime[:json])
      request.format = :json
    end
  end

  def model_class
    controller_name.classify.constantize
  end

  def resource_name             # params[] key used by client
    controller_name.singularize
  end

  def table_name
    controller_name
  end

  def find_object_for_update
    find_object_by_uuid(with_lock: true)
  end

  def find_object_by_uuid(with_lock: false)
    if params[:id] and params[:id].match(/\D/)
      params[:uuid] = params.delete :id
    end
    @where = { uuid: params[:uuid] }
    @offset = 0
    @limit = 1
    @orders = []
    @filters = []
    @objects = nil
    find_objects_for_index
    if with_lock
      @object = @objects.lock.first
    else
      @object = @objects.first
    end
  end

  def nullable_attributes
    []
  end

  # Go code may send empty values (ie: empty string instead of NULL) that
  # should be translated to NULL on the database.
  def set_nullable_attrs_to_null
    nullify_attrs(resource_attrs.to_hash).each do |k, v|
      resource_attrs[k] = v
    end
  end

  def nullify_attrs(a = {})
    new_attrs = a.to_hash.symbolize_keys
    (new_attrs.keys & nullable_attributes).each do |attr|
      val = new_attrs[attr]
      if (val.class == Integer && val == 0) || (val.class == String && val == "")
        new_attrs[attr] = nil
      end
    end
    return new_attrs
  end

  def reload_object_before_update
    # This is necessary to prevent an ActiveRecord::ReadOnlyRecord
    # error when updating an object which was retrieved using a join.
    if @object.andand.readonly?
      @object = model_class.find_by_uuid(@objects.first.uuid)
    end
  end

  def load_json_value(hash, key, must_be_class=nil)
    return if hash[key].nil?

    val = hash[key]
    if val.is_a? ActionController::Parameters
      val = val.to_unsafe_hash
    elsif val.is_a? String
      val = SafeJSON.load(val)
      hash[key] = val
    end
    # When assigning a Hash to an ActionController::Parameters and then
    # retrieve it, we get another ActionController::Parameters instead of
    # a Hash. This doesn't happen with other types. This is why 'val' is
    # being used to do type checking below.
    if must_be_class and !val.is_a? must_be_class
      raise TypeError.new("parameter #{key.to_s} must be a #{must_be_class.to_s}")
    end
  end

  def self.accept_attribute_as_json(attr, must_be_class=nil)
    before_action lambda { accept_attribute_as_json attr, must_be_class }
  end
  accept_attribute_as_json :properties, Hash
  accept_attribute_as_json :info, Hash
  def accept_attribute_as_json(attr, must_be_class)
    if params[resource_name] and [Hash, ActionController::Parameters].include?(resource_attrs.class)
      if resource_attrs[attr].is_a? Hash
        # Convert symbol keys to strings (in hashes provided by
        # resource_attrs)
        resource_attrs[attr] = resource_attrs[attr].
          with_indifferent_access.to_hash
      else
        load_json_value(resource_attrs, attr, must_be_class)
      end
    end
  end

  def self.accept_param_as_json(key, must_be_class=nil)
    prepend_before_action lambda { load_json_value(params, key, must_be_class) }
  end
  accept_param_as_json :reader_tokens, Array

  def object_list(model_class:)
    if @objects.respond_to?(:except)
      limit_database_read(model_class: model_class)
    end
    list = {
      :kind  => "arvados##{(@response_resource_name || resource_name).camelize(:lower)}List",
      :etag => "",
      :self_link => "",
      :offset => @offset,
      :limit => @limit,
      :items => @objects.as_api_response(nil, {select: @select})
    }
    if @extra_included
      list[:included] = @extra_included.as_api_response(nil, {select: @select})
    end
    case params[:count]
    when nil, '', 'exact'
      if @objects.respond_to? :except
        list[:items_available] = @objects.
          except(:limit).except(:offset).
          count(@distinct ? :id : '*')
      end
    when 'none'
    else
      raise ArgumentError.new("count parameter must be 'exact' or 'none'")
    end
    list
  end

  def render_list
    send_json object_list(model_class: self.model_class)
  end

  def remote_ip
    # Caveat: this is highly dependent on the proxy setup. YMMV.
    if request.headers.key?('HTTP_X_REAL_IP') then
      # We're behind a reverse proxy
      @remote_ip = request.headers['HTTP_X_REAL_IP']
    else
      # Hopefully, we are not!
      @remote_ip = request.env['REMOTE_ADDR']
    end
  end

  def load_required_parameters
    (self.class.send "_#{params[:action]}_requires_parameters" rescue {}).
      each do |key, info|
      if info[:required] and not params.include?(key)
        raise ArgumentError.new("#{key} parameter is required")
      elsif info[:type] == 'boolean'
        # Make sure params[key] is either true or false -- not a
        # string, not nil, etc.
        if not params.include?(key)
          params[key] = info[:default] || false
        elsif [false, 'false', '0', 0].include? params[key]
          params[key] = false
        elsif [true, 'true', '1', 1].include? params[key]
          params[key] = true
        else
          raise TypeError.new("#{key} parameter must be a boolean, true or false")
        end
      end
    end
    true
  end

  def self._create_requires_parameters
    {
      select: {
        type: 'array',
        description: "Attributes of the new object to return in the response.",
        required: false,
      },
      ensure_unique_name: {
        type: "boolean",
        description: "Adjust name to ensure uniqueness instead of returning an error on (owner_uuid, name) collision.",
        location: "query",
        required: false,
        default: false
      },
      cluster_id: {
        type: 'string',
        description: "Create object on a remote federated cluster instead of the current one.",
        location: "query",
        required: false,
      },
    }
  end

  def self._update_requires_parameters
    {
      select: {
        type: 'array',
        description: "Attributes of the updated object to return in the response.",
        required: false,
      },
    }
  end

  def self._show_requires_parameters
    {
      select: {
        type: 'array',
        description: "Attributes of the object to return in the response.",
        required: false,
      },
    }
  end

  def self._index_requires_parameters
    {
      filters: { type: 'array', required: false },
      where: { type: 'object', required: false },
      order: { type: 'array', required: false },
      select: {
        type: 'array',
        description: "Attributes of each object to return in the response.",
        required: false,
      },
      distinct: { type: 'boolean', required: false, default: false },
      limit: { type: 'integer', required: false, default: DEFAULT_LIMIT },
      offset: { type: 'integer', required: false, default: 0 },
      count: { type: 'string', required: false, default: 'exact' },
      cluster_id: {
        type: 'string',
        description: "List objects on a remote federated cluster instead of the current one.",
        location: "query",
        required: false,
      },
      bypass_federation: {
        type: 'boolean',
        required: false,
        description: 'bypass federation behavior, list items from local instance database only'
      }
    }
  end

  def render *opts
    if opts.first
      response = opts.first[:json]
      if response.is_a?(Hash) &&
          params[:_profile] &&
          Thread.current[:request_starttime]
        response[:_profile] = {
          request_time: Time.now - Thread.current[:request_starttime]
        }
      end
    end
    super(*opts)
  end
end
