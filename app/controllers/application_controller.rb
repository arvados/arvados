class ApplicationController < ActionController::Base
  protect_from_forgery
  before_filter :find_object_by_uuid, :except => [:index, :render_error, :render_not_found]

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
