class ApplicationController < ActionController::Base
  include CurrentApiClient

  protect_from_forgery
  before_filter :uncamelcase_params_hash_keys
  around_filter :thread_with_auth_info, :except => [:render_error, :render_not_found]
  before_filter :find_object_by_uuid, :except => :index

  before_filter :remote_ip
  before_filter :login_required, :except => :render_not_found

  before_filter :catch_redirect_hint

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

  def index
    @objects ||= model_class.
      joins("LEFT JOIN metadata permissions ON permissions.head=#{table_name}.owner AND permissions.tail=#{model_class.sanitize current_user.uuid} AND permissions.metadata_class='permission'").
      where("?=? OR #{table_name}.owner=? OR #{table_name}.uuid=? OR permissions.head IS NOT NULL",
            true, current_user.is_admin,
            current_user.uuid, current_user.uuid)
    if params[:where]
      where = params[:where]
      where = JSON.parse(where) if where.is_a?(String)
      conditions = ['1=1']
      where.each do |attr,value|
        if (!value.nil? and
            attr.to_s.match(/^[a-z][_a-z0-9]+$/) and
            model_class.columns.collect(&:name).index(attr))
          if value.is_a? Array
            conditions[0] << " and #{table_name}.#{attr} in (?)"
            conditions << value
          else
            conditions[0] << " and #{table_name}.#{attr}=?"
            conditions << value
          end
        end
      end
      if conditions.length > 1
        conditions[0].sub!(/^1=1 and /, '')
        @objects = @objects.
          where(*conditions)
      end
    end
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
    @attrs = params[resource_name]
    if @attrs.nil?
      raise "no #{resource_name} (or #{resource_name.camelcase(:lower)}) provided with request #{params.inspect}"
    end
    if @attrs.class == String
      @attrs = uncamelcase_hash_keys(JSON.parse @attrs)
    end
    @object = model_class.new @attrs
    @object.save
    show
  end

  def update
    @attrs = params[resource_name]
    if @attrs.is_a? String
      @attrs = uncamelcase_hash_keys(JSON.parse @attrs)
    end
    @object.update_attributes @attrs
    show
  end

  protected

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
      if params[:api_token]
        api_client_auth = ApiClientAuthorization.
          includes(:api_client, :user).
          where('api_token=?', params[:api_token]).
          first
        if api_client_auth
          session[:user_id] = api_client_auth.user.id
          session[:api_client_uuid] = api_client_auth.api_client.uuid
          user = api_client_auth.user
          api_client = api_client_auth.api_client
        end
      elsif session[:user_id]
        user = User.find(session[:user_id]) rescue nil
        api_client = ApiClient.
          where('uuid=?',session[:api_client_uuid]).
          first rescue nil
      end
      Thread.current[:api_client_trusted] = session[:api_client_trusted]
      Thread.current[:api_client_ip_address] = remote_ip
      Thread.current[:api_client] = api_client
      Thread.current[:user] = user
      yield
    ensure
      Thread.current[:api_client_trusted] = nil
      Thread.current[:api_client_ip_address] = nil
      Thread.current[:api_client_uuid] = nil
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
        params[resource_name][attr] = JSON.parse params[resource_name][attr]
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
      :kind  => "orvos##{resource_name}List",
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
