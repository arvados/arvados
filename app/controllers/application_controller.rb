class ApplicationController < ActionController::Base
  protect_from_forgery
  before_filter :find_object_by_uuid, :except => [:index, :render_error, :render_not_found]
  around_filter :thread_with_api_token, :except => [:render_error, :render_not_found]

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

  def error(opts)
    respond_to do |f|
      f.html { render opts.merge(controller: 'application', action: 'error') }
      f.json { render opts.merge(json: {success: false, errors: @errors}) }
    end
  end

  def render_error(e)
    logger.error e.inspect
    logger.error e.backtrace.collect { |x| x + "\n" }.join('') if e.backtrace
    if @object and @object.errors and @object.errors.full_messages
      @errors = @object.errors.full_messages
    else
      @errors = [e.inspect]
    end
    self.error status: 422
  end

  def render_not_found(e=ActionController::RoutingError.new("Path not found"))
    logger.error e.inspect
    @errors = ["Path not found"]
    self.error status: 404
  end


  def index
    @objects ||= model_class.all
    respond_to do |f|
      f.json { render json: @objects }
    end
  end

  def show
    if !@object
      render_not_found("object not found")
    end
    respond_to do |f|
      f.json { render json: @object }
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

  def thread_with_api_token
    begin
      if params[:api_token]
        Thread.current[:orvos_api_token] = params[:api_token]
        # Before copying the token into session[], do a simple API
        # call to verify its authenticity.
        if verify_api_token
          session[:orvos_api_token] = params[:api_token]
          if !request.format.json? and request.method == 'GET'
            # Repeat this request with api_token in the (new) session
            # cookie instead of the query string.  This prevents API
            # tokens from appearing in (and being inadvisedly copied
            # and pasted from) browser Location bars.
            redirect_to request.fullpath.sub(%r{([&\?]api_token=)[^&\?]*}, '')
          else
            yield
          end
        else
          @errors = ['Could not verify API token.']
          self.error status: 401
        end
      elsif session[:orvos_api_token]
        # In this case, the token must have already verified at some
        # point, although it might have been revoked since.  TODO:
        # graceful failure if the token is revoked.
        Thread.current[:orvos_api_token] = session[:orvos_api_token]
        yield
      else
        respond_to do |f|
          f.html {
            redirect_to $orvos_api_client.orvos_login_url(return_to: request.url)
          }
          f.json {
            @errors = ['No API token supplied -- can\'t really do anything.']
            self.error status: 422
          }
        end
      end
    ensure
      # Remove token in case this Thread is used for anything else.
      Thread.current[:orvos_api_token] = nil
    end
  end

  def verify_api_token
    Metadatum.where(uuid: 'the-philosophers-stone').size rescue false
  end
end
