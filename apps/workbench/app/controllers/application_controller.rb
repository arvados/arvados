# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

class ApplicationController < ActionController::Base
  include ArvadosApiClientHelper
  include ApplicationHelper

  respond_to :html, :json, :js
  protect_from_forgery

  ERROR_ACTIONS = [:render_error, :render_not_found]

  around_action :thread_clear
  around_action :set_current_request_id
  around_action :set_thread_api_token
  # Methods that don't require login should
  #   skip_around_action :require_thread_api_token
  around_action :require_thread_api_token, except: ERROR_ACTIONS
  before_action :ensure_arvados_api_exists, only: [:index, :show]
  before_action :set_cache_buster
  before_action :accept_uuid_as_id_param, except: ERROR_ACTIONS
  before_action :check_user_agreements, except: ERROR_ACTIONS
  before_action :check_user_profile, except: ERROR_ACTIONS
  before_action :load_filters_and_paging_params, except: ERROR_ACTIONS
  before_action :find_object_by_uuid, except: [:create, :index, :choose] + ERROR_ACTIONS
  theme :select_theme

  begin
    rescue_from(ActiveRecord::RecordNotFound,
                ActionController::RoutingError,
                AbstractController::ActionNotFound,
                with: :render_not_found)
    rescue_from(Exception,
                ActionController::UrlGenerationError,
                with: :render_exception)
  end

  def set_cache_buster
    response.headers["Cache-Control"] = "no-cache, no-store, max-age=0, must-revalidate"
    response.headers["Pragma"] = "no-cache"
    response.headers["Expires"] = "Fri, 01 Jan 1990 00:00:00 GMT"
  end

  def unprocessable(message=nil)
    @errors ||= []

    @errors << message if message
    render_error status: 422
  end

  def render_error(opts={})
    # Helpers can rely on the presence of @errors to know they're
    # being used in an error page.
    @errors ||= []
    opts[:status] ||= 500
    respond_to do |f|
      # json must come before html here, so it gets used as the
      # default format when js is requested by the client. This lets
      # ajax:error callback parse the response correctly, even though
      # the browser can't.
      f.json { render opts.merge(json: {success: false, errors: @errors}) }
      f.html { render({action: 'error'}.merge(opts)) }
      f.all { render({action: 'error', formats: 'text'}.merge(opts)) }
    end
  end

  def render_exception(e)
    logger.error e.inspect
    logger.error e.backtrace.collect { |x| x + "\n" }.join('') if e.backtrace
    err_opts = {status: 422}
    if e.is_a?(ArvadosApiClient::ApiError)
      err_opts.merge!(action: 'api_error', locals: {api_error: e})
      @errors = e.api_response[:errors]
    elsif @object.andand.errors.andand.full_messages.andand.any?
      @errors = @object.errors.full_messages
    else
      @errors = [e.to_s]
    end
    # Make user information available on the error page, falling back to the
    # session cache if the API server is unavailable.
    begin
      load_api_token(session[:arvados_api_token])
    rescue ArvadosApiClient::ApiError
      unless session[:user].nil?
        begin
          Thread.current[:user] = User.new(session[:user])
        rescue ArvadosApiClient::ApiError
          # This can happen if User's columns are unavailable.  Nothing to do.
        end
      end
    end
    # Preload projects trees for the template.  If that's not doable, set empty
    # trees so error page rendering can proceed.  (It's easier to rescue the
    # exception here than in a template.)
    unless current_user.nil?
      begin
        my_starred_projects current_user, 'project'
        build_my_wanted_projects_tree current_user
      rescue ArvadosApiClient::ApiError
        # Fall back to the default-setting code later.
      end
    end
    @starred_projects ||= []
    @my_wanted_projects_tree ||= []
    render_error(err_opts)
  end

  def render_not_found(e=ActionController::RoutingError.new("Path not found"))
    logger.error e.inspect
    @errors = ["Path not found"]
    set_thread_api_token do
      self.render_error(action: '404', status: 404)
    end
  end

  # params[:order]:
  #
  # The order can be left empty to allow it to default.
  # Or it can be a comma separated list of real database column names, one per model.
  # Column names should always be qualified by a table name and a direction is optional, defaulting to asc
  # (e.g. "collections.name" or "collections.name desc").
  # If a column name is specified, that table will be sorted by that column.
  # If there are objects from different models that will be shown (such as in Pipelines and processes tab),
  # then a sort column name can optionally be specified for each model, passed as an comma-separated list (e.g. "jobs.script, pipeline_instances.name")
  # Currently only one sort column name and direction can be specified for each model.
  def load_filters_and_paging_params
    if params[:order].blank?
      @order = 'created_at desc'
    elsif params[:order].is_a? Array
      @order = params[:order]
    else
      begin
        @order = JSON.load(params[:order])
      rescue
        @order = params[:order].split(',')
      end
    end
    @order = [@order] unless @order.is_a? Array

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
      elsif filters.is_a? Array
        filters = filters.collect do |filter|
          if filter.is_a? String
            # Accept filters[]=["foo","=","bar"]
            Oj.load filter
          else
            # Accept filters=[["foo","=","bar"]]
            filter
          end
        end
      end
      # After this, params[:filters] can be trusted to be an array of arrays:
      params[:filters] = filters
      @filters += filters
    end
  end

  def find_objects_for_index
    @objects ||= model_class
    @objects = @objects.filter(@filters).limit(@limit).offset(@offset)
    @objects.fetch_multiple_pages(false)
  end

  def render_index
    respond_to do |f|
      f.json {
        if params[:partial]
          @next_page_href = next_page_href(partial: params[:partial], filters: @filters.to_json)
          render json: {
            content: render_to_string(partial: "show_#{params[:partial]}",
                                      formats: [:html]),
            next_page_href: @next_page_href
          }
        else
          render json: @objects
        end
      }
      f.html {
        if params[:tab_pane]
          render_pane params[:tab_pane]
        else
          render
        end
      }
      f.js { render }
    end
  end

  helper_method :render_pane
  def render_pane tab_pane, opts={}
    render_opts = {
      partial: 'show_' + tab_pane.downcase,
      locals: {
        comparable: self.respond_to?(:compare),
        objects: @objects,
        tab_pane: tab_pane
      }.merge(opts[:locals] || {})
    }
    if opts[:to_string]
      render_to_string render_opts
    else
      render render_opts
    end
  end

  def ensure_arvados_api_exists
    if model_class.is_a?(Class) && model_class < ArvadosBase && !model_class.api_exists?(params['action'].to_sym)
      @errors = ["#{params['action']} method is not supported for #{params['controller']}"]
      return render_error(status: 404)
    end
  end

  def index
    @objects = nil if !defined?(@objects)
    find_objects_for_index if !@objects
    render_index
  end

  helper_method :next_page_offset
  def next_page_offset objects=nil
    if !objects
      objects = @objects
    end
    if objects.respond_to?(:result_offset) and
        objects.respond_to?(:result_limit)
      next_offset = objects.result_offset + objects.result_limit
      if objects.respond_to?(:items_available) and (next_offset < objects.items_available)
        next_offset
      elsif @objects.results.size > 0 and (params[:count] == 'none' or
           (params[:controller] == 'search' and params[:action] == 'choose'))
        last_object_class = @objects.last.class
        if params['last_object_class'].nil? or params['last_object_class'] == last_object_class.to_s
          next_offset
        else
          @objects.select{|obj| obj.class == last_object_class}.size
        end
      else
        nil
      end
    end
  end

  helper_method :next_page_href
  def next_page_href with_params={}
    if next_page_offset
      url_for with_params.merge(offset: next_page_offset)
    end
  end

  helper_method :next_page_filters
  def next_page_filters nextpage_operator
    next_page_filters = @filters.reject do |attr, op, val|
      (attr == 'created_at' and op == nextpage_operator) or
      (attr == 'uuid' and op == 'not in')
    end

    if @objects.any?
      last_created_at = @objects.last.created_at

      last_uuids = []
      @objects.each do |obj|
        last_uuids << obj.uuid if obj.created_at.eql?(last_created_at)
      end

      next_page_filters += [['created_at', nextpage_operator, last_created_at]]
      next_page_filters += [['uuid', 'not in', last_uuids]]
    end

    next_page_filters
  end

  def show
    if !@object
      return render_not_found("object not found")
    end
    respond_to do |f|
      f.json do
        extra_attrs = { href: url_for(action: :show, id: @object) }
        @object.textile_attributes.each do |textile_attr|
          extra_attrs.merge!({ "#{textile_attr}Textile" => view_context.render_markup(@object.attributes[textile_attr]) })
        end
        render json: @object.attributes.merge(extra_attrs)
      end
      f.html {
        if params['tab_pane']
          render_pane(if params['tab_pane'].is_a? Hash then params['tab_pane']["name"] else params['tab_pane'] end)
        elsif request.request_method.in? ['GET', 'HEAD']
          render
        else
          redirect_to (params[:return_to] ||
                       polymorphic_url(@object,
                                       anchor: params[:redirect_to_anchor]))
        end
      }
      f.js { render }
    end
  end

  def redirect_to uri, *args
    if request.xhr?
      if not uri.is_a? String
        uri = polymorphic_url(uri)
      end
      render json: {href: uri}
    else
      super
    end
  end

  def choose
    @objects = nil if !defined?(@objects)
    params[:limit] ||= 40
    respond_to do |f|
      if params[:partial]
        f.json {
          find_objects_for_index if !@objects
          render json: {
            content: render_to_string(partial: "choose_rows.html",
                                      formats: [:html]),
            next_page_href: next_page_href(partial: params[:partial])
          }
        }
      end
      f.js {
        find_objects_for_index if !@objects
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
    if @updates.is_a? ActionController::Parameters
      @updates = @updates.to_unsafe_hash
    end
    @updates.keys.each do |attr|
      if @object.send(attr).is_a? Hash
        if @updates[attr].is_a? String
          @updates[attr] = Oj.load @updates[attr]
        end
        if params[:merge] || params["merge_#{attr}".to_sym]
          # Merge provided Hash with current Hash, instead of
          # replacing.
          if @updates[attr].is_a? ActionController::Parameters
            @updates[attr] = @updates[attr].to_unsafe_hash
          end
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
    @object ||= model_class.new @new_resource_attrs, params["options"]

    if @object.save
      show
    else
      render_error status: 422
    end
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
        @object.name = ""
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
    Thread.current[:user]
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

  def set_share_links
    @user_is_manager = false
    @share_links = []

    if @object.uuid != current_user.andand.uuid
      begin
        @share_links = Link.permissions_for(@object)
        @user_is_manager = true
      rescue ArvadosApiClient::AccessForbiddenException,
        ArvadosApiClient::NotFoundException
      end
    end
  end

  def share_with
    if not params[:uuids].andand.any?
      @errors = ["No user/group UUIDs specified to share with."]
      return render_error(status: 422)
    end
    results = {"success" => [], "errors" => []}
    params[:uuids].each do |shared_uuid|
      begin
        Link.create(tail_uuid: shared_uuid, link_class: "permission",
                    name: "can_read", head_uuid: @object.uuid)
      rescue ArvadosApiClient::ApiError => error
        error_list = error.api_response.andand[:errors]
        if error_list.andand.any?
          results["errors"] += error_list.map { |e| "#{shared_uuid}: #{e}" }
        else
          error_code = error.api_status || "Bad status"
          results["errors"] << "#{shared_uuid}: #{error_code} response"
        end
      else
        results["success"] << shared_uuid
      end
    end
    if results["errors"].empty?
      results.delete("errors")
      status = 200
    else
      status = 422
    end
    respond_to do |f|
      f.json { render(json: results, status: status) }
    end
  end

  helper_method :is_starred
  def is_starred
    links = Link.where(tail_uuid: current_user.uuid,
               head_uuid: @object.uuid,
               link_class: 'star').with_count("none")

    return links.andand.any?
  end

  protected

  helper_method :strip_token_from_path
  def strip_token_from_path(path)
    path.sub(/([\?&;])api_token=[^&;]*[&;]?/, '\1')
  end

  def redirect_to_login
    if request.xhr? or request.format.json?
      @errors = ['You are not logged in. Most likely your session has timed out and you need to log in again.']
      render_error status: 401
    elsif request.method.in? ['GET', 'HEAD']
      redirect_to arvados_api_client.arvados_login_url(return_to: strip_token_from_path(request.url))
    else
      flash[:error] = "Either you are not logged in, or your session has timed out. I can't automatically log you in and re-attempt this request."
      redirect_to :back
    end
    false  # For convenience to return from callbacks
  end

  def using_specific_api_token(api_token, opts={})
    start_values = {}
    [:arvados_api_token, :user].each do |key|
      start_values[key] = Thread.current[key]
    end
    if opts.fetch(:load_user, true)
      load_api_token(api_token)
    else
      Thread.current[:arvados_api_token] = api_token
      Thread.current[:user] = nil
    end
    begin
      yield
    ensure
      start_values.each_key { |key| Thread.current[key] = start_values[key] }
    end
  end


  def accept_uuid_as_id_param
    if params[:id] and params[:id].match(/\D/)
      params[:uuid] = params.delete :id
    end
  end

  def find_object_by_uuid
    begin
      if not model_class
        @object = nil
      elsif params[:uuid].nil? or params[:uuid].empty?
        @object = nil
      elsif not params[:uuid].is_a?(String)
        @object = model_class.where(uuid: params[:uuid]).first
      elsif (model_class != Link and
             resource_class_for_uuid(params[:uuid]) == Link)
        @name_link = Link.find(params[:uuid])
        @object = model_class.find(@name_link.head_uuid)
      else
        @object = model_class.find(params[:uuid])
        load_preloaded_objects [@object]
      end
    rescue ArvadosApiClient::NotFoundException, ArvadosApiClient::NotLoggedInException, RuntimeError => error
      if error.is_a?(RuntimeError) and (error.message !~ /^argument to find\(/)
        raise
      end
      render_not_found(error)
      return false
    end
  end

  def thread_clear
    load_api_token(nil)
    Rails.cache.delete_matched(/^request_#{Thread.current.object_id}_/)
    yield
    Rails.cache.delete_matched(/^request_#{Thread.current.object_id}_/)
  end

  def set_current_request_id
    response.headers['X-Request-Id'] =
      Thread.current[:request_id] =
      "req-" + Random::DEFAULT.rand(2**128).to_s(36)[0..19]
    yield
    Thread.current[:request_id] = nil
  end

  def append_info_to_payload(payload)
    super
    payload[:request_id] = response.headers['X-Request-Id']
  end

  # Set up the thread with the given API token and associated user object.
  def load_api_token(new_token)
    Thread.current[:arvados_api_token] = new_token
    if new_token.nil?
      Thread.current[:user] = nil
    else
      Thread.current[:user] = User.current
    end
  end

  # If there's a valid api_token parameter, set up the session with that
  # user's information.  Return true if the method redirects the request
  # (usually a post-login redirect); false otherwise.
  def setup_user_session
    return false unless params[:api_token]
    Thread.current[:arvados_api_token] = params[:api_token]
    begin
      user = User.current
    rescue ArvadosApiClient::NotLoggedInException
      false  # We may redirect to login, or not, based on the current action.
    else
      session[:arvados_api_token] = params[:api_token]
      # If we later have trouble contacting the API server, we still want
      # to be able to render basic user information in the UI--see
      # render_exception above.  We store that in the session here.  This is
      # not intended to be used as a general-purpose cache.  See #2891.
      session[:user] = {
        uuid: user.uuid,
        email: user.email,
        first_name: user.first_name,
        last_name: user.last_name,
        is_active: user.is_active,
        is_admin: user.is_admin,
        prefs: user.prefs
      }

      if !request.format.json? and request.method.in? ['GET', 'HEAD']
        # Repeat this request with api_token in the (new) session
        # cookie instead of the query string.  This prevents API
        # tokens from appearing in (and being inadvisedly copied
        # and pasted from) browser Location bars.
        redirect_to strip_token_from_path(request.fullpath)
        true
      else
        false
      end
    ensure
      Thread.current[:arvados_api_token] = nil
    end
  end

  # Save the session API token in thread-local storage, and yield.
  # This method also takes care of session setup if the request
  # provides a valid api_token parameter.
  # If a token is unavailable or expired, the block is still run, with
  # a nil token.
  def set_thread_api_token
    if Thread.current[:arvados_api_token]
      yield   # An API token has already been found - pass it through.
      return
    elsif setup_user_session
      return  # A new session was set up and received a response.
    end

    begin
      load_api_token(session[:arvados_api_token])
      yield
    rescue ArvadosApiClient::NotLoggedInException
      # If we got this error with a token, it must've expired.
      # Retry the request without a token.
      unless Thread.current[:arvados_api_token].nil?
        load_api_token(nil)
        yield
      end
    ensure
      # Remove token in case this Thread is used for anything else.
      load_api_token(nil)
    end
  end

  # Redirect to login/welcome if client provided expired API token (or
  # none at all)
  def require_thread_api_token
    if Thread.current[:arvados_api_token]
      yield
    elsif session[:arvados_api_token]
      # Expired session. Clear it before refreshing login so that,
      # if this login procedure fails, we end up showing the "please
      # log in" page instead of getting stuck in a redirect loop.
      session.delete :arvados_api_token
      redirect_to_login
    elsif request.xhr?
      # If we redirect to the welcome page, the browser will handle
      # the 302 by itself and the client code will end up rendering
      # the "welcome" page in some content area where it doesn't make
      # sense. Instead, we send 401 ("authenticate and try again" or
      # "display error", depending on how smart the client side is).
      @errors = ['You are not logged in.']
      render_error status: 401
    else
      redirect_to welcome_users_path(return_to: request.fullpath)
    end
  end

  def ensure_current_user_is_admin
    if not current_user
      @errors = ['Not logged in']
      render_error status: 401
    elsif not current_user.is_admin
      @errors = ['Permission denied']
      render_error status: 403
    end
  end

  helper_method :unsigned_user_agreements
  def unsigned_user_agreements
    @signed_ua_uuids ||= UserAgreement.signatures.map &:head_uuid
    @unsigned_user_agreements ||= UserAgreement.all.map do |ua|
      if not @signed_ua_uuids.index ua.uuid
        Collection.find(ua.uuid)
      end
    end.compact
  end

  def check_user_agreements
    if current_user && !current_user.is_active
      if not current_user.is_invited
        return redirect_to inactive_users_path(return_to: request.fullpath)
      end
      if unsigned_user_agreements.empty?
        # No agreements to sign. Perhaps we just need to ask?
        current_user.activate
        if !current_user.is_active
          logger.warn "#{current_user.uuid.inspect}: " +
            "No user agreements to sign, but activate failed!"
        end
      end
      if !current_user.is_active
        redirect_to user_agreements_path(return_to: request.fullpath)
      end
    end
    true
  end

  def check_user_profile
    return true if !current_user
    if request.method.downcase != 'get' || params[:partial] ||
       params[:tab_pane] || params[:action_method] ||
       params[:action] == 'setup_popup'
      return true
    end

    if missing_required_profile?
      redirect_to profile_user_path(current_user.uuid, return_to: request.fullpath)
    end
    true
  end

  helper_method :missing_required_profile?
  def missing_required_profile?
    missing_required = false

    profile_config = Rails.configuration.Workbench.UserProfileFormFields
    if current_user && !profile_config.empty?
      current_user_profile = current_user.prefs[:profile]
      profile_config.each do |k, entry|
        if entry[:Required]
          if !current_user_profile ||
             !current_user_profile[k] ||
             current_user_profile[k].empty?
            missing_required = true
            break
          end
        end
      end
    end

    missing_required
  end

  def select_theme
    return Rails.configuration.Workbench.Theme
  end

  @@notification_tests = []

  @@notification_tests.push lambda { |controller, current_user|
    return nil if Rails.configuration.Services.WebShell.ExternalURL != URI("")
    AuthorizedKey.limit(1).with_count('none').where(authorized_user_uuid: current_user.uuid).each do
      return nil
    end
    return lambda { |view|
      view.render partial: 'notifications/ssh_key_notification'
    }
  }

  @@notification_tests.push lambda { |controller, current_user|
    Collection.limit(1).with_count('none').where(created_by: current_user.uuid).each do
      return nil
    end
    return lambda { |view|
      view.render partial: 'notifications/collections_notification'
    }
  }

  @@notification_tests.push lambda { |controller, current_user|
    if PipelineInstance.api_exists?(:index)
      PipelineInstance.limit(1).with_count('none').where(created_by: current_user.uuid).each do
        return nil
      end
    else
      return nil
    end
    return lambda { |view|
      view.render partial: 'notifications/pipelines_notification'
    }
  }

  helper_method :user_notifications
  def user_notifications
    @errors = nil if !defined?(@errors)
    return [] if @errors or not current_user.andand.is_active or not Rails.configuration.Workbench.ShowUserNotifications
    @notifications ||= @@notification_tests.map do |t|
      t.call(self, current_user)
    end.compact
  end

  helper_method :all_projects
  def all_projects
    @all_projects ||= Group.
      filter([['group_class','IN',['project','filter']]]).order('name')
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
    (Job.limit(10) |
     PipelineInstance.limit(10).with_count("none")).
      sort_by do |x|
      (x.finished_at || x.started_at rescue nil) || x.modified_at || x.created_at
    end.reverse
  end

  helper_method :running_pipelines
  def running_pipelines
    pi = PipelineInstance.order(["started_at asc", "created_at asc"]).with_count("none").filter([["state", "in", ["RunningOnServer", "RunningOnClient"]]])
    jobs = {}
    pi.each do |pl|
      pl.components.each do |k,v|
        if v.is_a? Hash and v[:job]
          jobs[v[:job][:uuid]] = {}
        end
      end
    end

    if jobs.keys.any?
      Job.filter([["uuid", "in", jobs.keys]]).with_count("none").each do |j|
        jobs[j[:uuid]] = j
      end

      pi.each do |pl|
        pl.components.each do |k,v|
          if v.is_a? Hash and v[:job]
            v[:job] = jobs[v[:job][:uuid]]
          end
        end
      end
    end

    pi
  end

  helper_method :recent_processes
  def recent_processes lim
    lim = 12 if lim.nil?

    procs = {}
    if PipelineInstance.api_exists?(:index)
      cols = %w(uuid owner_uuid created_at modified_at pipeline_template_uuid name state started_at finished_at)
      pipelines = PipelineInstance.select(cols).limit(lim).order(["created_at desc"]).with_count("none")
      pipelines.results.each { |pi| procs[pi] = pi.created_at }
    end

    crs = ContainerRequest.limit(lim).with_count("none").order(["created_at desc"]).filter([["requesting_container_uuid", "=", nil]])
    crs.results.each { |c| procs[c] = c.created_at }

    Hash[procs.sort_by {|key, value| value}].keys.reverse.first(lim)
  end

  helper_method :recent_collections
  def recent_collections lim
    c = Collection.limit(lim).with_count("none").order(["modified_at desc"]).results
    own = {}
    Group.filter([["uuid", "in", c.map(&:owner_uuid)]]).with_count("none").each do |g|
      own[g[:uuid]] = g
    end
    {collections: c, owners: own}
  end

  helper_method :my_starred_projects
  def my_starred_projects user, group_class
    return if defined?(@starred_projects) && @starred_projects
    links = Link.filter([['owner_uuid', 'in', ["#{Rails.configuration.ClusterID}-j7d0g-publicfavorites", user.uuid]],
                         ['link_class', '=', 'star'],
                         ['head_uuid', 'is_a', 'arvados#group']]).with_count("none").select(%w(head_uuid))
    uuids = links.collect { |x| x.head_uuid }
    if group_class == ""
      starred_projects = Group.filter([['uuid', 'in', uuids]]).order('name').with_count("none")
    else
      starred_projects = Group.filter([['uuid', 'in', uuids],['group_class', '=', group_class]]).order('name').with_count("none")
    end
    @starred_projects = starred_projects.results
  end

  # If there are more than 200 projects that are readable by the user,
  # build the tree using only the top 200+ projects owned by the user,
  # from the top three levels.
  # That is: get toplevel projects under home, get subprojects of
  # these projects, and so on until we hit the limit.
  def my_wanted_projects(user, page_size=100)
    return @my_wanted_projects if defined?(@my_wanted_projects) && @my_wanted_projects

    from_top = []
    uuids = [user.uuid]
    depth = 0
    @too_many_projects = false
    @reached_level_limit = false
    while from_top.size <= page_size*2
      current_level = Group.filter([['group_class','IN',['project','filter']],
                                    ['owner_uuid', 'in', uuids]])
                      .order('name').limit(page_size*2)
      break if current_level.results.size == 0
      @too_many_projects = true if current_level.items_available > current_level.results.size
      from_top.concat current_level.results
      uuids = current_level.results.collect(&:uuid)
      depth += 1
      if depth >= 3
        @reached_level_limit = true
        break
      end
    end
    @my_wanted_projects = from_top
  end

  helper_method :my_wanted_projects_tree
  def my_wanted_projects_tree(user, page_size=100)
    build_my_wanted_projects_tree(user, page_size)
    [@my_wanted_projects_tree, @too_many_projects, @reached_level_limit]
  end

  def build_my_wanted_projects_tree(user, page_size=100)
    return @my_wanted_projects_tree if defined?(@my_wanted_projects_tree) && @my_wanted_projects_tree

    parent_of = {user.uuid => 'me'}
    my_wanted_projects(user, page_size).each do |ob|
      parent_of[ob.uuid] = ob.owner_uuid
    end
    children_of = {false => [], 'me' => [user]}
    my_wanted_projects(user, page_size).each do |ob|
      if ob.owner_uuid != user.uuid and
          not parent_of.has_key? ob.owner_uuid
        parent_of[ob.uuid] = false
      end
      children_of[parent_of[ob.uuid]] ||= []
      children_of[parent_of[ob.uuid]] << ob
    end
    buildtree = lambda do |chldrn_of, root_uuid=false|
      tree = {}
      chldrn_of[root_uuid].andand.each do |ob|
        tree[ob] = buildtree.call(chldrn_of, ob.uuid)
      end
      tree
    end
    sorted_paths = lambda do |tree, depth=0|
      paths = []
      tree.keys.sort_by { |ob|
        ob.is_a?(String) ? ob : ob.friendly_link_name
      }.each do |ob|
        paths << {object: ob, depth: depth}
        paths += sorted_paths.call tree[ob], depth+1
      end
      paths
    end
    @my_wanted_projects_tree =
      sorted_paths.call buildtree.call(children_of, 'me')
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
      # Halt if a group ownership loop is detected. API should refuse
      # to produce this state, but it could still arise from a race
      # condition when group ownership changes between our find()
      # queries.
      break if crumbs.collect(&:uuid).include? current.uuid

      if current.is_a?(Group) and current.group_class == 'project'
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
    if @object.is_a? Group and @object.group_class == 'project'
      @object.uuid
    elsif @name_link.andand.tail_uuid
      @name_link.tail_uuid
    elsif @object and resource_class_for_uuid(@object.owner_uuid) == Group
      @object.owner_uuid
    else
      nil
    end
  end

  # helper method to get links for given object or uuid
  helper_method :links_for_object
  def links_for_object object_or_uuid
    raise ArgumentError, 'No input argument' unless object_or_uuid
    preload_links_for_objects([object_or_uuid])
    uuid = object_or_uuid.is_a?(String) ? object_or_uuid : object_or_uuid.uuid
    @all_links_for[uuid] ||= []
  end

  # helper method to preload links for given objects and uuids
  helper_method :preload_links_for_objects
  def preload_links_for_objects objects_and_uuids
    @all_links_for ||= {}

    raise ArgumentError, 'Argument is not an array' unless objects_and_uuids.is_a? Array
    return @all_links_for if objects_and_uuids.empty?

    uuids = objects_and_uuids.collect { |x| x.is_a?(String) ? x : x.uuid }

    # if already preloaded for all of these uuids, return
    if not uuids.select { |x| @all_links_for[x].nil? }.any?
      return @all_links_for
    end

    uuids.each do |x|
      @all_links_for[x] = []
    end

    # TODO: make sure we get every page of results from API server
    Link.filter([['head_uuid', 'in', uuids]]).with_count("none").each do |link|
      @all_links_for[link.head_uuid] << link
    end
    @all_links_for
  end

  # helper method to get a certain number of objects of a specific type
  # this can be used to replace any uses of: "dataclass.limit(n)"
  helper_method :get_n_objects_of_class
  def get_n_objects_of_class dataclass, size
    @objects_map_for ||= {}

    raise ArgumentError, 'Argument is not a data class' unless dataclass.is_a? Class and dataclass < ArvadosBase
    raise ArgumentError, 'Argument is not a valid limit size' unless (size && size>0)

    # if the objects_map_for has a value for this dataclass, and the
    # size used to retrieve those objects is equal, return it
    size_key = "#{dataclass.name}_size"
    if @objects_map_for[dataclass.name] && @objects_map_for[size_key] &&
        (@objects_map_for[size_key] == size)
      return @objects_map_for[dataclass.name]
    end

    @objects_map_for[size_key] = size
    @objects_map_for[dataclass.name] = dataclass.limit(size)
  end

  # helper method to get collections for the given uuid
  helper_method :collections_for_object
  def collections_for_object uuid
    raise ArgumentError, 'No input argument' unless uuid
    preload_collections_for_objects([uuid])
    @all_collections_for[uuid] ||= []
  end

  # helper method to preload collections for the given uuids
  helper_method :preload_collections_for_objects
  def preload_collections_for_objects uuids
    @all_collections_for ||= {}

    raise ArgumentError, 'Argument is not an array' unless uuids.is_a? Array
    return @all_collections_for if uuids.empty?

    # if already preloaded for all of these uuids, return
    if not uuids.select { |x| @all_collections_for[x].nil? }.any?
      return @all_collections_for
    end

    uuids.each do |x|
      @all_collections_for[x] = []
    end

    # TODO: make sure we get every page of results from API server
    Collection.where(uuid: uuids).with_count("none").each do |collection|
      @all_collections_for[collection.uuid] << collection
    end
    @all_collections_for
  end

  # helper method to get log collections for the given log
  helper_method :log_collections_for_object
  def log_collections_for_object log
    raise ArgumentError, 'No input argument' unless log

    preload_log_collections_for_objects([log])

    uuid = log
    fixup = /([a-f0-9]{32}\+\d+)(\+?.*)/.match(log)
    if fixup && fixup.size>1
      uuid = fixup[1]
    end

    @all_log_collections_for[uuid] ||= []
  end

  # helper method to preload collections for the given uuids
  helper_method :preload_log_collections_for_objects
  def preload_log_collections_for_objects logs
    @all_log_collections_for ||= {}

    raise ArgumentError, 'Argument is not an array' unless logs.is_a? Array
    return @all_log_collections_for if logs.empty?

    uuids = []
    logs.each do |log|
      fixup = /([a-f0-9]{32}\+\d+)(\+?.*)/.match(log)
      if fixup && fixup.size>1
        uuids << fixup[1]
      else
        uuids << log
      end
    end

    # if already preloaded for all of these uuids, return
    if not uuids.select { |x| @all_log_collections_for[x].nil? }.any?
      return @all_log_collections_for
    end

    uuids.each do |x|
      @all_log_collections_for[x] = []
    end

    # TODO: make sure we get every page of results from API server
    Collection.where(uuid: uuids).with_count("none").each do |collection|
      @all_log_collections_for[collection.uuid] << collection
    end
    @all_log_collections_for
  end

  # Helper method to get one collection for the given portable_data_hash
  # This is used to determine if a pdh is readable by the current_user
  helper_method :collection_for_pdh
  def collection_for_pdh pdh
    raise ArgumentError, 'No input argument' unless pdh
    preload_for_pdhs([pdh])
    @all_pdhs_for[pdh] ||= []
  end

  # Helper method to preload one collection each for the given pdhs
  # This is used to determine if a pdh is readable by the current_user
  helper_method :preload_for_pdhs
  def preload_for_pdhs pdhs
    @all_pdhs_for ||= {}

    raise ArgumentError, 'Argument is not an array' unless pdhs.is_a? Array
    return @all_pdhs_for if pdhs.empty?

    # if already preloaded for all of these pdhs, return
    if not pdhs.select { |x| @all_pdhs_for[x].nil? }.any?
      return @all_pdhs_for
    end

    pdhs.each do |x|
      @all_pdhs_for[x] = []
    end

    Collection.select(%w(portable_data_hash)).where(portable_data_hash: pdhs).distinct().with_count("none").each do |collection|
      @all_pdhs_for[collection.portable_data_hash] << collection
    end
    @all_pdhs_for
  end

  # helper method to get object of a given dataclass and uuid
  helper_method :object_for_dataclass
  def object_for_dataclass dataclass, uuid, by_attr=nil
    raise ArgumentError, 'No input argument dataclass' unless (dataclass && uuid)
    preload_objects_for_dataclass(dataclass, [uuid], by_attr)
    @objects_for[uuid]
  end

  # helper method to preload objects for given dataclass and uuids
  helper_method :preload_objects_for_dataclass
  def preload_objects_for_dataclass dataclass, uuids, by_attr=nil
    @objects_for ||= {}

    raise ArgumentError, 'Argument is not a data class' unless dataclass.is_a? Class
    raise ArgumentError, 'Argument is not an array' unless uuids.is_a? Array

    return @objects_for if uuids.empty?

    # if already preloaded for all of these uuids, return
    if not uuids.select { |x| !@objects_for.include?(x) }.any?
      return @objects_for
    end

    # preset all uuids to nil
    uuids.each do |x|
      @objects_for[x] = nil
    end
    if by_attr and ![:uuid, :name].include?(by_attr)
      raise ArgumentError, "Preloading only using lookups by uuid or name are supported: #{by_attr}"
    elsif by_attr and by_attr == :name
      dataclass.where(name: uuids).each do |obj|
        @objects_for[obj.name] = obj
      end
    else
      key_prefix = "request_#{Thread.current.object_id}_#{dataclass.to_s}_"
      dataclass.where(uuid: uuids).each do |obj|
        @objects_for[obj.uuid] = obj
        if dataclass == Collection
          # The collecions#index defaults to "all attributes except manifest_text"
          # Hence, this object is not suitable for preloading the find() cache.
        else
          Rails.cache.write(key_prefix + obj.uuid, obj.as_json)
        end
      end
    end
    @objects_for
  end

  # helper method to load objects that are already preloaded
  helper_method :load_preloaded_objects
  def load_preloaded_objects objs
    @objects_for ||= {}
    objs.each do |obj|
      @objects_for[obj.uuid] = obj
    end
  end

  # helper method to get the names of collection files selected
  helper_method :selected_collection_files
  def selected_collection_files params
    link_uuids, coll_ids = params["selection"].partition do |sel_s|
      ArvadosBase::resource_class_for_uuid(sel_s) == Link
    end

    unless link_uuids.empty?
      Link.select([:head_uuid]).where(uuid: link_uuids).with_count("none").each do |link|
        if ArvadosBase::resource_class_for_uuid(link.head_uuid) == Collection
          coll_ids << link.head_uuid
        end
      end
    end

    uuids = []
    pdhs = []
    source_paths = Hash.new { |hash, key| hash[key] = [] }
    coll_ids.each do |coll_id|
      if m = CollectionsHelper.match(coll_id)
        key = m[1] + m[2]
        pdhs << key
        source_paths[key] << m[4]
      elsif m = CollectionsHelper.match_uuid_with_optional_filepath(coll_id)
        key = m[1]
        uuids << key
        source_paths[key] << m[4]
      end
    end

    unless pdhs.empty?
      Collection.where(portable_data_hash: pdhs.uniq).with_count("none").
          select([:uuid, :portable_data_hash]).each do |coll|
        unless source_paths[coll.portable_data_hash].empty?
          uuids << coll.uuid
          source_paths[coll.uuid] = source_paths.delete(coll.portable_data_hash)
        end
      end
    end

    [uuids, source_paths]
  end

  def wiselinks_layout
    'body'
  end
end
