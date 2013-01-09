class ApplicationController < ActionController::Base
  protect_from_forgery
  before_filter :uncamelcase_params_hash_keys
  before_filter :find_object_by_uuid, :except => :index

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
      raise "no #{resource_name} provided with request #{params.inspect}"
    end
    if @attrs.class == String
      @attrs = uncamelcase_hash_keys(JSON.parse @attrs)
    end
    @object = model_class.new @attrs
    @object.save
    show
  end

  protected

  def model_class
    controller_name.classify.constantize
  end

  def resource_name             # params[] key used by client
    controller_name.classify.camelcase(:lower)
  end

  def find_object_by_uuid
    logger.info params.inspect
    @object = model_class.where('uuid=?', params[:uuid]).first
  end

  def uncamelcase_params_hash_keys
    self.params = uncamelcase_hash_keys(params)
  end

  def uncamelcase_hash_keys(h)
    if h.is_a? Hash
      nh = Hash.new
      h.each do |k,v|
        if k.class == String
          nk = k.underscore
        elsif k.class == Symbol
          nk = k.to_s.underscore.to_sym
        else
          nk = k
        end
        nh[nk] = uncamelcase_hash_keys(v)
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
