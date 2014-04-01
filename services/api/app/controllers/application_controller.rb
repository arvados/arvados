class ApplicationController < ActionController::Base
  include CurrentApiClient

  respond_to :json
  protect_from_forgery
  around_filter :thread_with_auth_info, :except => [:render_error, :render_not_found]

  before_filter :remote_ip
  before_filter :require_auth_scope_all, :except => :render_not_found
  before_filter :catch_redirect_hint

  before_filter :load_where_param, :only => :index
  before_filter :load_filters_param, :only => :index
  before_filter :find_objects_for_index, :only => :index
  before_filter :find_object_by_uuid, :except => [:index, :create,
                                                  :render_error,
                                                  :render_not_found]
  before_filter :reload_object_before_update, :only => :update
  before_filter :render_404_if_no_object, except: [:index, :create,
                                                   :render_error,
                                                   :render_not_found]

  attr_accessor :resource_attrs

  def index
    @objects.uniq!(&:id)
    if params[:eager] and params[:eager] != '0' and params[:eager] != 0 and params[:eager] != ''
      @objects.each(&:eager_load_associations)
    end
    render_list
  end

  def show
    render json: @object.as_api_response
  end

  def create
    @object = model_class.new resource_attrs
    if @object.save
      show
    else
      raise "Save failed"
    end
  end

  def update
    attrs_to_update = resource_attrs.reject { |k,v|
      [:kind, :etag, :href].index k
    }
    if @object.update_attributes attrs_to_update
      show
    else
      raise "Update failed"
    end
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

  begin
    rescue_from Exception,
    :with => :render_error
    rescue_from ActiveRecord::RecordNotFound,
    :with => :render_not_found
    rescue_from ActionController::RoutingError,
    :with => :render_not_found
    rescue_from ActionController::UnknownController,
    :with => :render_not_found
    rescue_from AbstractController::ActionNotFound,
    :with => :render_not_found
    rescue_from ArvadosModel::PermissionDeniedError,
    :with => :render_error
  end

  def render_404_if_no_object
    render_not_found "Object not found" if !@object
  end

  def render_error(e)
    logger.error e.inspect
    if e.respond_to? :backtrace and e.backtrace
      logger.error e.backtrace.collect { |x| x + "\n" }.join('')
    end
    if @object and @object.errors and @object.errors.full_messages and not @object.errors.full_messages.empty?
      errors = @object.errors.full_messages
    else
      errors = [e.inspect]
    end
    status = e.respond_to?(:http_status) ? e.http_status : 422
    render json: { errors: errors }, status: status
  end

  def render_not_found(e=ActionController::RoutingError.new("Path not found"))
    logger.error e.inspect
    render json: { errors: ["Path not found"] }, status: 404
  end

  protected

  def load_where_param
    if params[:where].nil? or params[:where] == ""
      @where = {}
    elsif params[:where].is_a? Hash
      @where = params[:where]
    elsif params[:where].is_a? String
      begin
        @where = Oj.load(params[:where])
        raise unless @where.is_a? Hash
      rescue
        raise ArgumentError.new("Could not parse \"where\" param as an object")
      end
    end
    @where = @where.with_indifferent_access
  end

  def load_filters_param
    if params[:filters].is_a? Array
      @filters = params[:filters]
    elsif params[:filters].is_a? String and !params[:filters].empty?
      begin
        @filters = Oj.load params[:filters]
        raise unless @filters.is_a? Array
      rescue
        raise ArgumentError.new("Could not parse \"filters\" param as an array")
      end
    end
  end

  def find_objects_for_index
    @objects ||= model_class.readable_by(current_user)
    apply_where_limit_order_params
  end

  def apply_where_limit_order_params
    if @filters.is_a? Array and @filters.any?
      cond_out = []
      param_out = []
      @filters.each do |attr, operator, operand|
        if !model_class.searchable_columns.index attr.to_s
          raise ArgumentError.new("Invalid attribute '#{attr}' in condition")
        end
        case operator.downcase
        when '=', '<', '<=', '>', '>=', 'like'
          if operand.is_a? String
            cond_out << "#{table_name}.#{attr} #{operator} ?"
            if (# any operator that operates on value rather than
                # representation:
                operator.match(/[<=>]/) and
                model_class.attribute_column(attr).type == :datetime)
              operand = Time.parse operand
            end
            param_out << operand
          end
        when 'in'
          if operand.is_a? Array
            cond_out << "#{table_name}.#{attr} IN (?)"
            param_out << operand
          end
        end
      end
      if cond_out.any?
        @objects = @objects.where(cond_out.join(' AND '), *param_out)
      end
    end
    if @where.is_a? Hash and @where.any?
      conditions = ['1=1']
      @where.each do |attr,value|
        if attr == :any
          if value.is_a?(Array) and
              value.length == 2 and
              value[0] == 'contains' and
              model_class.columns.collect(&:name).index('name') then
            ilikes = []
            model_class.searchable_columns.each do |column|
              ilikes << "#{table_name}.#{column} ilike ?"
              conditions << "%#{value[1]}%"
            end
            if ilikes.any?
              conditions[0] << ' and (' + ilikes.join(' or ') + ')'
            end
          end
        elsif attr.to_s.match(/^[a-z][_a-z0-9]+$/) and
            model_class.columns.collect(&:name).index(attr.to_s)
          if value.nil?
            conditions[0] << " and #{table_name}.#{attr} is ?"
            conditions << nil
          elsif value.is_a? Array
            if value[0] == 'contains' and value.length == 2
              conditions[0] << " and #{table_name}.#{attr} like ?"
              conditions << "%#{value[1]}%"
            else
              conditions[0] << " and #{table_name}.#{attr} in (?)"
              conditions << value
            end
          elsif value.is_a? String or value.is_a? Fixnum or value == true or value == false
            conditions[0] << " and #{table_name}.#{attr}=?"
            conditions << value
          elsif value.is_a? Hash
            # Not quite the same thing as "equal?" but better than nothing?
            value.each do |k,v|
              if v.is_a? String
                conditions[0] << " and #{table_name}.#{attr} ilike ?"
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

    if params[:limit]
      begin
        @limit = params[:limit].to_i
      rescue
        raise ArgumentError.new("Invalid value for limit parameter")
      end
    else
      @limit = 100
    end
    @objects = @objects.limit(@limit)

    orders = []

    if params[:offset]
      begin
        @objects = @objects.offset(params[:offset].to_i)
        @offset = params[:offset].to_i
      rescue
        raise ArgumentError.new("Invalid value for limit parameter")
      end
    else
      @offset = 0
    end      

    orders = []
    if params[:order]
      params[:order].split(',').each do |order|
        attr, direction = order.strip.split " "
        direction ||= 'asc'
        if attr.match /^[a-z][_a-z0-9]+$/ and
            model_class.columns.collect(&:name).index(attr) and
            ['asc','desc'].index direction.downcase
          orders << "#{table_name}.#{attr} #{direction.downcase}"
        end
      end
    end
    if orders.empty?
      orders << "#{table_name}.modified_at desc"
    end
    @objects = @objects.order(orders.join ", ")
  end

  def resource_attrs
    return @attrs if @attrs
    @attrs = params[resource_name]
    if @attrs.is_a? String
      @attrs = Oj.load @attrs, symbol_keys: true
    end
    unless @attrs.is_a? Hash
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
    @attrs = @attrs.symbolize_keys if @attrs.is_a? HashWithIndifferentAccess
    @attrs
  end

  # Authentication
  def require_login
    if current_user
      true
    else
      respond_to do |format|
        format.json {
          render :json => { errors: ['Not logged in'] }.to_json, status: 401
        }
        format.html  {
          redirect_to '/auth/joshid'
        }
      end
      false
    end
  end

  def admin_required
    unless current_user and current_user.is_admin
      render :json => { errors: ['Forbidden'] }.to_json, status: 403
    end
  end

  def require_auth_scope_all
    require_login and require_auth_scope(['all'])
  end

  def require_auth_scope(ok_scopes)
    unless current_api_client_auth_has_scope(ok_scopes)
      render :json => { errors: ['Forbidden'] }.to_json, status: 403
    end
  end

  def thread_with_auth_info
    Thread.current[:request_starttime] = Time.now
    Thread.current[:api_url_base] = root_url.sub(/\/$/,'') + '/arvados/v1'
    begin
      user = nil
      api_client = nil
      api_client_auth = nil
      supplied_token =
        params[:api_token] ||
        params[:oauth_token] ||
        request.headers["Authorization"].andand.match(/OAuth2 ([a-z0-9]+)/).andand[1]
      if supplied_token
        api_client_auth = ApiClientAuthorization.
          includes(:api_client, :user).
          where('api_token=? and (expires_at is null or expires_at > CURRENT_TIMESTAMP)', supplied_token).
          first
        if api_client_auth.andand.user
          session[:user_id] = api_client_auth.user.id
          session[:api_client_uuid] = api_client_auth.api_client.andand.uuid
          session[:api_client_authorization_id] = api_client_auth.id
          user = api_client_auth.user
          api_client = api_client_auth.api_client
        end
      elsif session[:user_id]
        user = User.find(session[:user_id]) rescue nil
        api_client = ApiClient.
          where('uuid=?',session[:api_client_uuid]).
          first rescue nil
        if session[:api_client_authorization_id] then
          api_client_auth = ApiClientAuthorization.
            find session[:api_client_authorization_id]
        end
      end
      Thread.current[:api_client_ip_address] = remote_ip
      Thread.current[:api_client_authorization] = api_client_auth
      Thread.current[:api_client_uuid] = api_client.andand.uuid
      Thread.current[:api_client] = api_client
      Thread.current[:user] = user
      if api_client_auth
        api_client_auth.last_used_at = Time.now
        api_client_auth.last_used_by_ip_address = remote_ip
        api_client_auth.save validate: false
      end
      yield
    ensure
      Thread.current[:api_client_ip_address] = nil
      Thread.current[:api_client_authorization] = nil
      Thread.current[:api_client_uuid] = nil
      Thread.current[:api_client] = nil
      Thread.current[:user] = nil
    end
  end
  # /Authentication

  def model_class
    controller_name.classify.constantize
  end

  def resource_name             # params[] key used by client
    controller_name.singularize
  end

  def table_name
    controller_name
  end

  def find_object_by_uuid
    if params[:id] and params[:id].match /\D/
      params[:uuid] = params.delete :id
    end
    @where = { uuid: params[:uuid] }
    find_objects_for_index
    @object = @objects.first
  end

  def reload_object_before_update
    # This is necessary to prevent an ActiveRecord::ReadOnlyRecord
    # error when updating an object which was retrieved using a join.
    if @object.andand.readonly?
      @object = model_class.find_by_uuid(@objects.first.uuid)
    end
  end

  def self.accept_attribute_as_json(attr, force_class=nil)
    before_filter lambda { accept_attribute_as_json attr, force_class }
  end
  accept_attribute_as_json :properties, Hash
  accept_attribute_as_json :info, Hash
  def accept_attribute_as_json(attr, force_class)
    if params[resource_name] and resource_attrs.is_a? Hash
      if resource_attrs[attr].is_a? String
        resource_attrs[attr] = Oj.load(resource_attrs[attr],
                                       symbol_keys: false)
        if force_class and !resource_attrs[attr].is_a? force_class
          raise TypeError.new("#{resource_name}[#{attr.to_s}] must be a #{force_class.to_s}")
        end
      elsif resource_attrs[attr].is_a? Hash
        # Convert symbol keys to strings (in hashes provided by
        # resource_attrs)
        resource_attrs[attr] = resource_attrs[attr].
          with_indifferent_access.to_hash
      end
    end
  end

  def render_list
    @object_list = {
      :kind  => "arvados##{(@response_resource_name || resource_name).camelize(:lower)}List",
      :etag => "",
      :self_link => "",
      :offset => @offset,
      :limit => @limit,
      :items => @objects.as_api_response(nil)
    }
    if @objects.respond_to? :except
      @object_list[:items_available] = @objects.
        except(:limit).except(:offset).
        count(:id, distinct: true)
    end
    render json: @object_list
  end

  def remote_ip
    # Caveat: this is highly dependent on the proxy setup. YMMV.
    if request.headers.has_key?('HTTP_X_REAL_IP') then
      # We're behind a reverse proxy
      @remote_ip = request.headers['HTTP_X_REAL_IP']
    else
      # Hopefully, we are not!
      @remote_ip = request.env['REMOTE_ADDR']
    end
  end

  def self._index_requires_parameters
    {
      filters: { type: 'array', required: false },
      where: { type: 'object', required: false },
      order: { type: 'string', required: false }
    }
  end
  
  def client_accepts_plain_text_stream
    (request.headers['Accept'].split(' ') &
     ['text/plain', '*/*']).count > 0
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
    super *opts
  end
end
