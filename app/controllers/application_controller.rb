class ApplicationController < ActionController::Base
  protect_from_forgery
  before_filter :uncamelcase_params_hash_keys
  before_filter :find_object_by_uuid, :except => :index

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
    if @object and @object.errors and @object.errors.full_messages
      errors = @object.errors.full_messages
    else
      errors = [e.inspect]
    end
    render json: { errors: errors }, status: 422
  end

  def render_not_found
    render json: { errors: ["Path not found"] }, status: 401
  end

  def index
    @objects ||= model_class.all
    render_list
  end

  def show
    render json: @object
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

  def model_class
    controller_name.classify.constantize
  end

  def resource_name             # params[] key used by client
    controller_name.singularize
  end

  def find_object_by_uuid
    logger.info params.inspect
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
      :items => @objects.map { |x| x }
    }
    render json: @object_list
  end
end
