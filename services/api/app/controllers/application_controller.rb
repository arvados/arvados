class ApplicationController < ActionController::Base
  include CurrentApiClient

  protect_from_forgery
  before_filter :uncamelcase_params_hash_keys
  around_filter :thread_with_auth_info, :except => [:render_error, :render_not_found]

  before_filter :remote_ip
  before_filter :login_required, :except => :render_not_found
  before_filter :catch_redirect_hint

  before_filter :find_objects_for_index, :only => :index
  before_filter :find_object_by_uuid, :except => [:index, :create]

  attr_accessor :resource_attrs

  def index
    @objects.uniq!(&:id)
    if params[:eager] and params[:eager] != '0' and params[:eager] != 0 and params[:eager] != ''
      @objects.each(&:eager_load_associations)
    end
    render_list
  end

  def show
    if @object
      render json: @object.as_api_response(:superuser)
    else
      render_not_found("object not found")
    end
  end

  def create
    @object = model_class.new resource_attrs
    @object.save
    show
  end

  def update
    if @object.update_attributes resource_attrs
      show
    else
      render json: { errors: @object.errors.full_messages }, status: 422
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

  unless Rails.application.config.consider_all_requests_local
    rescue_from Exception,
    :with => :render_error
    rescue_from ActiveRecord::RecordNotFound,
    :with => :render_not_found
    rescue_from ActionController::RoutingError,
    :with => :render_not_found
    rescue_from ActionController::UnknownController,
    :with => :render_not_found
    rescue_from ActionController::UnknownAction,
    :with => :render_not_found
  end

  def render_error(e)
    logger.error e.inspect
    logger.error e.backtrace.collect { |x| x + "\n" }.join('') if e.backtrace
    if @object and @object.errors and @object.errors.full_messages
      errors = @object.errors.full_messages
    else
      errors = [e.inspect]
    end
    render json: { errors: errors }, status: 422
  end

  def render_not_found(e=ActionController::RoutingError.new("Path not found"))
    logger.error e.inspect
    render json: { errors: ["Path not found"] }, status: 404
  end

  protected

  def find_objects_for_index
    uuid_list = [current_user.uuid, *current_user.groups_i_can(:read)]
    sanitized_uuid_list = uuid_list.
      collect { |uuid| model_class.sanitize(uuid) }.join(', ')
    @objects ||= model_class.
      joins("LEFT JOIN links permissions ON permissions.head_uuid=#{table_name}.owner AND permissions.tail_uuid in (#{sanitized_uuid_list}) AND permissions.link_class='permission'").
      where("?=? OR #{table_name}.owner in (?) OR #{table_name}.uuid=? OR permissions.head_uuid IS NOT NULL",
            true, current_user.is_admin,
            uuid_list,
            current_user.uuid)
    @where = params[:where] || {}
    @where = Oj.load(@where) if @where.is_a?(String)
    if params[:where]
      conditions = ['1=1']
      @where.each do |attr,value|
        if attr == 'any'
          if value.is_a?(Array) and
              value[0] == 'contains' and
              model_class.columns.collect(&:name).index('name') then
            conditions[0] << " and #{table_name}.name ilike ?"
            conditions << "%#{value[1]}%"
          end
        elsif attr.to_s.match(/^[a-z][_a-z0-9]+$/) and
            model_class.columns.collect(&:name).index(attr)
          if value.nil?
            conditions[0] << " and #{table_name}.#{attr} is ?"
            conditions << nil
          elsif value.is_a? Array
            conditions[0] << " and #{table_name}.#{attr} in (?)"
            conditions << value
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
        @objects = @objects.limit(params[:limit].to_i)
      rescue
        raise ArgumentError.new("Invalid value for limit parameter")
      end
    else
      @objects = @objects.limit(100)
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
      @attrs = uncamelcase_hash_keys(Oj.load @attrs)
    end
    unless @attrs.is_a? Hash
      message = "No #{resource_name}"
      if resource_name.index('_')
        message << " (or #{resource_name.camelcase(:lower)})"
      end
      message << " hash provided with request"
      raise ArgumentError.new(message)
    end
    %w(created_at modified_by_client modified_by_user modified_at).each do |x|
      @attrs.delete x
    end
    @attrs
  end

  # Authentication
  def login_required
    if !current_user
      respond_to do |format|
        format.html  {
          redirect_to '/auth/joshid'
        }
        format.json {
          render :json => { errors: ['Not logged in'] }.to_json
        }
      end
    end
  end

  def thread_with_auth_info
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
          where('api_token=?', supplied_token).
          first
        if api_client_auth
          session[:user_id] = api_client_auth.user.id
          session[:api_client_uuid] = api_client_auth.api_client.uuid
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
      Thread.current[:api_client_trusted] = session[:api_client_trusted]
      Thread.current[:api_client_ip_address] = remote_ip
      Thread.current[:api_client_authorization] = api_client_auth
      Thread.current[:api_client_uuid] = api_client && api_client.uuid
      Thread.current[:api_client] = api_client
      Thread.current[:user] = user
      yield
    ensure
      Thread.current[:api_client_trusted] = nil
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
    @object = model_class.where('uuid=?', params[:uuid]).first
  end

  def self.accept_attribute_as_json(attr, force_class=nil)
    before_filter lambda { accept_attribute_as_json attr, force_class }
  end
  def accept_attribute_as_json(attr, force_class)
    if params[resource_name].is_a? Hash
      if params[resource_name][attr].is_a? String
        params[resource_name][attr] = Oj.load params[resource_name][attr]
        if force_class and !params[resource_name][attr].is_a? force_class
          raise TypeError.new("#{resource_name}[#{attr.to_s}] must be a #{force_class.to_s}")
        end
      end
    end
  end

  def uncamelcase_params_hash_keys
    self.params = uncamelcase_hash_keys(params)
  end
  def uncamelcase_hash_keys(h, max_depth=-1)
    if h.is_a? Hash and max_depth != 0
      nh = Hash.new
      h.each do |k,v|
        if k.class == String
          nk = k.underscore
        elsif k.class == Symbol
          nk = k.to_s.underscore.to_sym
        else
          nk = k
        end
        nh[nk] = uncamelcase_hash_keys(v, max_depth-1)
      end
      h.replace(nh)
    end
    h
  end

  def render_list
    @object_list = {
      :kind  => "arvados##{resource_name}List",
      :etag => "",
      :self_link => "",
      :next_page_token => "",
      :next_link => "",
      :items => @objects.as_api_response(:superuser)
    }
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
end
