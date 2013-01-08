class ApplicationController < ActionController::Base
  protect_from_forgery
  before_filter :uncamelcase_params_hash_keys
  before_filter :find_object_by_uuid

  protected

  def model_class
    controller_name.classify.constantize
  end

  def find_object_by_uuid
    logger.info params.inspect
    @object = model_class.where('uuid=?', params[:uuid]).first
  end

  def uncamelcase_params_hash_keys
    uncamelcase_hash_keys(params)
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
end
