class ApplicationController < ActionController::Base
  protect_from_forgery
  before_filter :find_object_by_uuid, :except => :index

  def index
    @objects ||= model_class.all
  end

  def show
    if !@object
      render_not_found("object not found")
    end
  end

  protected
    
  def model_class
    controller_name.classify.constantize
  end

  def find_object_by_uuid
    if params[:id] and params[:id].match /\D/
      params[:uuid] = params.delete :id
    end
    @object = model_class.where('uuid=?', params[:uuid]).first
  end
end
