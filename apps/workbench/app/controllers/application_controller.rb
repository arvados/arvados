class ApplicationController < ActionController::Base
  include ArvadosApiClientHelper
  include ApplicationHelper

  respond_to :html, :json, :js
  protect_from_forgery

  ERROR_ACTIONS = [:render_error, :render_not_found]

  around_filter :thread_clear
  around_filter :thread_with_mandatory_api_token, except: ERROR_ACTIONS
  around_filter :thread_with_optional_api_token
  before_filter :check_user_agreements, except: ERROR_ACTIONS
  before_filter :check_user_notifications, except: ERROR_ACTIONS
  before_filter :find_object_by_uuid, except: [:index] + ERROR_ACTIONS
  theme :select_theme

  begin
    rescue_from Exception,
    :with => :render_exception
    rescue_from ActiveRecord::RecordNotFound,
    :with => :render_not_found
    rescue_from ActionController::RoutingError,
    :with => :render_not_found
    rescue_from ActionController::UnknownController,
    :with => :render_not_found
    rescue_from ::AbstractController::ActionNotFound,
    :with => :render_not_found
  end

  def unprocessable(message=nil)
    @errors ||= []

    @errors << message if message
    render_error status: 422
  end

  def render_error(opts)
    opts = {status: 500}.merge opts
    respond_to do |f|
      # json must come before html here, so it gets used as the
      # default format when js is requested by the client. This lets
      # ajax:error callback parse the response correctly, even though
      # the browser can't.
      f.json { render opts.merge(json: {success: false, errors: @errors}) }
      f.html { render opts.merge(controller: 'application', action: 'error') }
    end
  end

  def render_exception(e)
    logger.error e.inspect
    logger.error e.backtrace.collect { |x| x + "\n" }.join('') if e.backtrace
    if @object.andand.errors.andand.full_messages.andand.any?
      @errors = @object.errors.full_messages
    else
      @errors = [e.to_s]
    end
    self.render_error status: 422
  end

  def render_not_found(e=ActionController::RoutingError.new("Path not found"))
    logger.error e.inspect
    @errors = ["Path not found"]
    self.render_error status: 404
  end

  def find_objects_for_index
    @limit ||= 200
    if params[:limit]
      @limit = params[:limit].to_i
    end

    @offset ||= 0
    if params[:offset]
      @offset = params[:offset].to_i
    end

    @filters ||= []
    if params[:filters]
      filters = params[:filters]
      if filters.is_a? String
        filters = Oj.load filters
      end
      @filters += filters
    end

    @objects ||= model_class
    @objects = @objects.filter(@filters).limit(@limit).offset(@offset)
  end

  helper_method :next_page_offset
  def next_page_offset
    if @objects.respond_to?(:result_offset) and
        @objects.respond_to?(:result_limit) and
        @objects.respond_to?(:items_available)
      next_offset = @objects.result_offset + @objects.result_limit
      if next_offset < @objects.items_available
        next_offset
      else
        nil
      end
    end
  end

  def index
    find_objects_for_index if !@objects
    respond_to do |f|
      f.json { render json: @objects }
      f.html { render }
      f.js { render }
    end
  end

  def show
    if !@object
      return render_not_found("object not found")
    end
    respond_to do |f|
      f.json { render json: @object.attributes.merge(href: url_for(@object)) }
      f.html {
        if request.method.in? ['GET', 'HEAD']
          render
        else
          redirect_to params[:return_to] || @object
        end
      }
      f.js { render }
    end
  end

  def choose
    params[:limit] ||= 20
    find_objects_for_index if !@objects
    respond_to do |f|
      if params[:partial]
        f.json {
          render json: {
            content: render_to_string(partial: "choose_rows.html",
                                      formats: [:html],
                                      locals: {
                                        multiple: params[:multiple]
                                      }),
            next_page_href: @next_page_href
          }
        }
      end
      f.js {
        render partial: 'choose', locals: {multiple: params[:multiple]}
      }
    end
  end

  def render_content
    if !@object
      return render_not_found("object not found")
    end
  end

  def new
    @object = model_class.new
  end

  def update
    @updates ||= params[@object.resource_param_name.to_sym]
    @updates.keys.each do |attr|
      if @object.send(attr).is_a? Hash
        if @updates[attr].is_a? String
          @updates[attr] = Oj.load @updates[attr]
        end
        if params[:merge] || params["merge_#{attr}".to_sym]
          # Merge provided Hash with current Hash, instead of
          # replacing.
          @updates[attr] = @object.send(attr).with_indifferent_access.
            deep_merge(@updates[attr].with_indifferent_access)
        end
      end
    end
    if @object.update_attributes @updates
      show
    else
      self.render_error status: 422
    end
  end

  def create
    @new_resource_attrs ||= params[model_class.to_s.underscore.singularize]
    @new_resource_attrs ||= {}
    @new_resource_attrs.reject! { |k,v| k.to_s == 'uuid' }
    @object ||= model_class.new @new_resource_attrs
    @object.save!
    show
  end

  # Clone the given object, merging any attribute values supplied as
  # with a create action.
  def copy
    @new_resource_attrs ||= params[model_class.to_s.underscore.singularize]
    @new_resource_attrs ||= {}
    @object = @object.dup
    @object.update_attributes @new_resource_attrs
    if not @new_resource_attrs[:name] and @object.respond_to? :name
      if @object.name and @object.name != ''
        @object.name = "Copy of #{@object.name}"
      else
        @object.name = "Copy of unnamed #{@object.class_for_display.downcase}"
      end
    end
    @object.save!
    show
  end

  def destroy
    if @object.destroy
      respond_to do |f|
        f.json { render json: @object }
        f.html {
          redirect_to(params[:return_to] || :back)
        }
        f.js { render }
      end
    else
      self.render_error status: 422
    end
  end

  def current_user
    if Thread.current[:arvados_api_token]
      Thread.current[:user] ||= User.current
    else
      logger.error "No API token in Thread"
      return nil
    end
  end

  def model_class
    controller_name.classify.constantize
  end

  def breadcrumb_page_name
    (@breadcrumb_page_name ||
     (@object.friendly_link_name if @object.respond_to? :friendly_link_name) ||
     action_name)
  end

  def index_pane_list
    %w(Recent)
  end

  def show_pane_list
    %w(Attributes Advanced)
  end

  protected

  def redirect_to_login
    respond_to do |f|
      f.html {
        if request.method.in? ['GET', 'HEAD']
          redirect_to arvados_api_client.arvados_login_url(return_to: request.url)
        else
          flash[:error] = "Either you are not logged in, or your session has timed out. I can't automatically log you in and re-attempt this request."
          redirect_to :back
        end
      }
      f.json {
        @errors = ['You do not seem to be logged in. You did not supply an API token with this request, and your session (if any) has timed out.']
        self.render_error status: 422
      }
    end
    false  # For convenience to return from callbacks
  end

  def using_specific_api_token(api_token)
    start_values = {}
    [:arvados_api_token, :user].each do |key|
      start_values[key] = Thread.current[key]
    end
    Thread.current[:arvados_api_token] = api_token
    Thread.current[:user] = nil
    begin
      yield
    ensure
      start_values.each_key { |key| Thread.current[key] = start_values[key] }
    end
  end

  def find_object_by_uuid
    if params[:id] and params[:id].match /\D/
      params[:uuid] = params.delete :id
    end
    if not model_class
      @object = nil
    elsif params[:uuid].is_a? String
      if params[:uuid].empty?
        @object = nil
      else
        if (model_class != Link and
            resource_class_for_uuid(params[:uuid]) == Link)
          @name_link = Link.find(params[:uuid])
          @object = model_class.find(@name_link.head_uuid)
        else
          @object = model_class.find(params[:uuid])
        end
      end
    else
      @object = model_class.where(uuid: params[:uuid]).first
    end
  end

  def thread_clear
    Thread.current[:arvados_api_token] = nil
    Thread.current[:user] = nil
    Rails.cache.delete_matched(/^request_#{Thread.current.object_id}_/)
    yield
    Rails.cache.delete_matched(/^request_#{Thread.current.object_id}_/)
  end

  def thread_with_api_token(login_optional = false)
    begin
      try_redirect_to_login = true
      if params[:api_token]
        try_redirect_to_login = false
        Thread.current[:arvados_api_token] = params[:api_token]
        # Before copying the token into session[], do a simple API
        # call to verify its authenticity.
        if verify_api_token
          session[:arvados_api_token] = params[:api_token]
          if !request.format.json? and request.method.in? ['GET', 'HEAD']
            # Repeat this request with api_token in the (new) session
            # cookie instead of the query string.  This prevents API
            # tokens from appearing in (and being inadvisedly copied
            # and pasted from) browser Location bars.
            redirect_to request.fullpath.sub(%r{([&\?]api_token=)[^&\?]*}, '')
          else
            yield
          end
        else
          @errors = ['Invalid API token']
          self.render_error status: 401
        end
      elsif session[:arvados_api_token]
        # In this case, the token must have already verified at some
        # point, but it might have been revoked since.  We'll try
        # using it, and catch the exception if it doesn't work.
        try_redirect_to_login = false
        Thread.current[:arvados_api_token] = session[:arvados_api_token]
        begin
          yield
        rescue ArvadosApiClient::NotLoggedInException
          try_redirect_to_login = true
        end
      else
        logger.debug "No token received, session is #{session.inspect}"
      end
      if try_redirect_to_login
        unless login_optional
          redirect_to_login
        else
          # login is optional for this route so go on to the regular controller
          Thread.current[:arvados_api_token] = nil
          yield
        end
      end
    ensure
      # Remove token in case this Thread is used for anything else.
      Thread.current[:arvados_api_token] = nil
    end
  end

  def thread_with_mandatory_api_token
    thread_with_api_token(true) do
      if Thread.current[:arvados_api_token]
        yield
      elsif session[:arvados_api_token]
        # Expired session. Clear it before refreshing login so that,
        # if this login procedure fails, we end up showing the "please
        # log in" page instead of getting stuck in a redirect loop.
        session.delete :arvados_api_token
        redirect_to_login
      else
        render 'users/welcome'
      end
    end
  end

  # This runs after thread_with_mandatory_api_token in the filter chain.
  def thread_with_optional_api_token
    if Thread.current[:arvados_api_token]
      # We are already inside thread_with_mandatory_api_token.
      yield
    else
      # We skipped thread_with_mandatory_api_token. Use the optional version.
      thread_with_api_token(true) do
        yield
      end
    end
  end

  def verify_api_token
    begin
      Link.where(uuid: 'just-verifying-my-api-token')
      true
    rescue ArvadosApiClient::NotLoggedInException
      false
    end
  end

  def ensure_current_user_is_admin
    unless current_user and current_user.is_admin
      @errors = ['Permission denied']
      self.render_error status: 401
    end
  end

  def check_user_agreements
    if current_user && !current_user.is_active
      if not current_user.is_invited
        return render 'users/inactive'
      end
      signatures = UserAgreement.signatures
      @signed_ua_uuids = UserAgreement.signatures.map &:head_uuid
      @required_user_agreements = UserAgreement.all.map do |ua|
        if not @signed_ua_uuids.index ua.uuid
          Collection.find(ua.uuid)
        end
      end.compact
      if @required_user_agreements.empty?
        # No agreements to sign. Perhaps we just need to ask?
        current_user.activate
        if !current_user.is_active
          logger.warn "#{current_user.uuid.inspect}: " +
            "No user agreements to sign, but activate failed!"
        end
      end
      if !current_user.is_active
        render 'user_agreements/index'
      end
    end
    true
  end

  def select_theme
    return Rails.configuration.arvados_theme
  end

  @@notification_tests = []

  @@notification_tests.push lambda { |controller, current_user|
    AuthorizedKey.limit(1).where(authorized_user_uuid: current_user.uuid).each do
      return nil
    end
    return lambda { |view|
      view.render partial: 'notifications/ssh_key_notification'
    }
  }

  #@@notification_tests.push lambda { |controller, current_user|
  #  Job.limit(1).where(created_by: current_user.uuid).each do
  #    return nil
  #  end
  #  return lambda { |view|
  #    view.render partial: 'notifications/jobs_notification'
  #  }
  #}

  @@notification_tests.push lambda { |controller, current_user|
    Collection.limit(1).where(created_by: current_user.uuid).each do
      return nil
    end
    return lambda { |view|
      view.render partial: 'notifications/collections_notification'
    }
  }

  @@notification_tests.push lambda { |controller, current_user|
    PipelineInstance.limit(1).where(created_by: current_user.uuid).each do
      return nil
    end
    return lambda { |view|
      view.render partial: 'notifications/pipelines_notification'
    }
  }

  def check_user_notifications
    @notification_count = 0
    @notifications = []

    if current_user
      @showallalerts = false
      @@notification_tests.each do |t|
        a = t.call(self, current_user)
        if a
          @notification_count += 1
          @notifications.push a
        end
      end
    end

    if @notification_count == 0
      @notification_count = ''
    end
  end

  helper_method :all_projects
  def all_projects
    @all_projects ||= Group.filter([['group_class','in',['project','folder']]])
  end

  helper_method :my_projects
  def my_projects
    return @my_projects if @my_projects
    @my_projects = []
    root_of = {}
    all_projects.each do |g|
      root_of[g.uuid] = g.owner_uuid
      @my_projects << g
    end
    done = false
    while not done
      done = true
      root_of = root_of.each_with_object({}) do |(child, parent), h|
        if root_of[parent]
          h[child] = root_of[parent]
          done = false
        else
          h[child] = parent
        end
      end
    end
    @my_projects = @my_projects.select do |g|
      root_of[g.uuid] == current_user.uuid
    end
  end

  helper_method :projects_shared_with_me
  def projects_shared_with_me
    my_project_uuids = my_projects.collect &:uuid
    all_projects.reject { |x| x.uuid.in? my_project_uuids }
  end

  helper_method :recent_jobs_and_pipelines
  def recent_jobs_and_pipelines
    in_my_projects = ['owner_uuid','in',my_projects.collect(&:uuid)]
    (Job.limit(10).filter([in_my_projects]) |
     PipelineInstance.limit(10).filter([in_my_projects])).
      sort_by do |x|
      x.finished_at || x.started_at || x.created_at rescue x.created_at
    end
  end

  helper_method :get_object
  def get_object uuid
    if @get_object.nil? and @objects
      @get_object = @objects.each_with_object({}) do |object, h|
        h[object.uuid] = object
      end
    end
    @get_object ||= {}
    @get_object[uuid]
  end

  helper_method :project_breadcrumbs
  def project_breadcrumbs
    crumbs = []
    current = @name_link || @object
    while current
      if current.is_a?(Group) and current.group_class.in?(['project','folder'])
        crumbs.prepend current
      end
      if current.is_a? Link
        current = Group.find?(current.tail_uuid)
      else
        current = Group.find?(current.owner_uuid)
      end
    end
    crumbs
  end

  helper_method :current_project_uuid
  def current_project_uuid
    if @object.is_a? Group and @object.group_class.in?(['project','folder'])
      @object.uuid
    elsif @name_link.andand.tail_uuid
      @name_link.tail_uuid
    elsif @object and resource_class_for_uuid(@object.owner_uuid) == Group
      @object.owner_uuid
    else
      nil
    end
  end
end
