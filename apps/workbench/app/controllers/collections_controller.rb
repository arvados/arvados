class CollectionsController < ApplicationController
  skip_around_filter(:require_thread_api_token,
                     only: [:show_file, :show_file_links])
  skip_before_filter(:find_object_by_uuid,
                     only: [:provenance, :show_file, :show_file_links])
  # We depend on show_file to display the user agreement:
  skip_before_filter :check_user_agreements, only: :show_file
  skip_before_filter :check_user_profile, only: :show_file

  RELATION_LIMIT = 5

  def show_pane_list
    %w(Files Provenance_graph Used_by Advanced)
  end

  def set_persistent
    case params[:value]
    when 'persistent', 'cache'
      persist_links = Link.filter([['owner_uuid', '=', current_user.uuid],
                                   ['link_class', '=', 'resources'],
                                   ['name', '=', 'wants'],
                                   ['tail_uuid', '=', current_user.uuid],
                                   ['head_uuid', '=', @object.uuid]])
      logger.debug persist_links.inspect
    else
      return unprocessable "Invalid value #{value.inspect}"
    end
    if params[:value] == 'persistent'
      if not persist_links.any?
        Link.create(link_class: 'resources',
                    name: 'wants',
                    tail_uuid: current_user.uuid,
                    head_uuid: @object.uuid)
      end
    else
      persist_links.each do |link|
        link.destroy || raise
      end
    end

    respond_to do |f|
      f.json { render json: @object }
    end
  end

  def choose
    # Find collections using default find_objects logic, then search for name
    # links, and preload any other links connected to the collections that are
    # found.
    # Name links will be obsolete when issue #3036 is merged,
    # at which point this entire custom #choose function can probably be
    # eliminated.

    params[:limit] ||= 40

    find_objects_for_index
    @collections = @objects

    @filters += [['link_class','=','name'],
                 ['head_uuid','is_a','arvados#collection']]

    @objects = Link
    find_objects_for_index

    @name_links = @objects

    @objects = Collection.
      filter([['uuid','in',@name_links.collect(&:head_uuid)]])

    preload_links_for_objects (@collections.to_a + @objects.to_a)
    super
  end

  def index
    # API server index doesn't return manifest_text by default, but our
    # callers want it unless otherwise specified.
    @select ||= Collection.columns.map(&:name)
    base_search = Collection.select(@select)
    if params[:search].andand.length.andand > 0
      tags = Link.where(any: ['contains', params[:search]])
      @collections = (base_search.where(uuid: tags.collect(&:head_uuid)) |
                      base_search.where(any: ['contains', params[:search]])).
        uniq { |c| c.uuid }
    else
      if params[:limit]
        limit = params[:limit].to_i
      else
        limit = 100
      end

      if params[:offset]
        offset = params[:offset].to_i
      else
        offset = 0
      end

      @collections = base_search.limit(limit).offset(offset)
    end
    @links = Link.limit(1000).
      where(head_uuid: @collections.collect(&:uuid))
    @collection_info = {}
    @collections.each do |c|
      @collection_info[c.uuid] = {
        tag_links: [],
        wanted: false,
        wanted_by_me: false,
        provenance: [],
        links: []
      }
    end
    @links.each do |link|
      @collection_info[link.head_uuid] ||= {}
      info = @collection_info[link.head_uuid]
      case link.link_class
      when 'tag'
        info[:tag_links] << link
      when 'resources'
        info[:wanted] = true
        info[:wanted_by_me] ||= link.tail_uuid == current_user.uuid
      when 'provenance'
        info[:provenance] << link.name
      end
      info[:links] << link
    end
    @request_url = request.url

    render_index
  end

  def show_file_links
    Thread.current[:reader_tokens] = [params[:reader_token]]
    return if false.equal?(find_object_by_uuid)
    render layout: false
  end

  def show_file
    # We pipe from arv-get to send the file to the user.  Before we start it,
    # we ask the API server if the file actually exists.  This serves two
    # purposes: it lets us return a useful status code for common errors, and
    # helps us figure out which token to provide to arv-get.
    coll = nil
    tokens = [Thread.current[:arvados_api_token], params[:reader_token]].compact
    usable_token = find_usable_token(tokens) do
      coll = Collection.find(params[:uuid])
    end
    if usable_token.nil?
      return  # Response already rendered.
    elsif params[:file].nil? or not coll.manifest.has_file?(params[:file])
      return render_not_found
    end
    opts = params.merge(arvados_api_token: usable_token)
    ext = File.extname(params[:file])
    self.response.headers['Content-Type'] =
      Rack::Mime::MIME_TYPES[ext] || 'application/octet-stream'
    self.response.headers['Content-Length'] = params[:size] if params[:size]
    self.response.headers['Content-Disposition'] = params[:disposition] if params[:disposition]
    self.response_body = file_enumerator opts
  end

  def sharing_scopes
    ["GET /arvados/v1/collections/#{@object.uuid}", "GET /arvados/v1/collections/#{@object.uuid}/", "GET /arvados/v1/keep_services/accessible"]
  end

  def search_scopes
    begin
      ApiClientAuthorization.filter([['scopes', '=', sharing_scopes]]).results
    rescue ArvadosApiClient::AccessForbiddenException
      nil
    end
  end

  def show
    return super if !@object
    if current_user
      jobs_with = lambda do |conds|
        Job.limit(RELATION_LIMIT).where(conds)
          .results.sort_by { |j| j.finished_at || j.created_at }
      end
      @output_of = jobs_with.call(output: @object.portable_data_hash)
      @log_of = jobs_with.call(log: @object.portable_data_hash)
      @project_links = Link.limit(RELATION_LIMIT).order("modified_at DESC")
        .where(head_uuid: @object.uuid, link_class: 'name').results
      project_hash = Group.where(uuid: @project_links.map(&:tail_uuid)).to_hash
      @projects = project_hash.values

      if @object.uuid.match /[0-9a-f]{32}/
        @same_pdh = Collection.filter([["portable_data_hash", "=", @object.portable_data_hash]])
        owners = @same_pdh.map {|s| s.owner_uuid}.to_a
        preload_objects_for_dataclass Group, owners
        preload_objects_for_dataclass User, owners
      end

      @permissions = Link.limit(RELATION_LIMIT).order("modified_at DESC")
        .where(head_uuid: @object.uuid, link_class: 'permission',
               name: 'can_read').results
      @logs = Log.limit(RELATION_LIMIT).order("created_at DESC")
        .where(object_uuid: @object.uuid).results
      @is_persistent = Link.limit(1)
        .where(head_uuid: @object.uuid, tail_uuid: current_user.uuid,
               link_class: 'resources', name: 'wants')
        .results.any?
      @search_sharing = search_scopes
    end

    if params["tab_pane"] == "Provenance_graph"
      @prov_svg = ProvenanceHelper::create_provenance_graph(@object.provenance, "provenance_svg",
                                                            {:request => request,
                                                              :direction => :bottom_up,
                                                              :combine_jobs => :script_only}) rescue nil
    end
    if params["tab_pane"] == "Used_by"
      @used_by_svg = ProvenanceHelper::create_provenance_graph(@object.used_by, "used_by_svg",
                                                               {:request => request,
                                                                 :direction => :top_down,
                                                                 :combine_jobs => :script_only,
                                                                 :pdata_only => true}) rescue nil
    end
    super
  end

  def sharing_popup
    @search_sharing = search_scopes
    respond_to do |format|
      format.html
      format.js
    end
  end

  helper_method :download_link

  def download_link
    collections_url + "/download/#{@object.uuid}/#{@search_sharing.first.api_token}/"
  end

  def share
    a = ApiClientAuthorization.create(scopes: sharing_scopes)
    @search_sharing = search_scopes
    render 'sharing_popup'
  end

  def unshare
    @search_sharing = search_scopes
    @search_sharing.each do |s|
      s.destroy
    end
    @search_sharing = search_scopes
    render 'sharing_popup'
  end

  protected

  def find_usable_token(token_list)
    # Iterate over every given token to make it the current token and
    # yield the given block.
    # If the block succeeds, return the token it used.
    # Otherwise, render an error response based on the most specific
    # error we encounter, and return nil.
    most_specific_error = [401]
    token_list.each do |api_token|
      begin
        using_specific_api_token(api_token) do
          yield
          return api_token
        end
      rescue ArvadosApiClient::ApiError => error
        if error.api_status >= most_specific_error.first
          most_specific_error = [error.api_status, error]
        end
      end
    end
    case most_specific_error.shift
    when 401, 403
      redirect_to_login
    when 404
      render_not_found(*most_specific_error)
    end
    return nil
  end

  def file_enumerator(opts)
    FileStreamer.new opts
  end

  class FileStreamer
    include ArvadosApiClientHelper
    def initialize(opts={})
      @opts = opts
    end
    def each
      return unless @opts[:uuid] && @opts[:file]

      env = Hash[ENV].dup

      require 'uri'
      u = URI.parse(arvados_api_client.arvados_v1_base)
      env['ARVADOS_API_HOST'] = "#{u.host}:#{u.port}"
      env['ARVADOS_API_TOKEN'] = @opts[:arvados_api_token]
      env['ARVADOS_API_HOST_INSECURE'] = "true" if Rails.configuration.arvados_insecure_https

      bytesleft = @opts[:size].andand.to_i || 2**16
      IO.popen([env, 'arv-get', "#{@opts[:uuid]}/#{@opts[:file]}"],
               'rb') do |io|
        while bytesleft > 0 && buf = io.read(bytesleft)
          # shrink the bytesleft count, if we were given a
          # maximum byte count to read
          if @opts.include? :size
            bytesleft = bytesleft - buf.length
          end
          yield buf
        end
        if @opts.include? :size and bytesleft == 0 then
          yield "Log truncated (exceeded #{@opts[:size]} bytes). To retrieve the full log: arv-get #{@opts[:uuid]}/#{@opts[:file]}"
        end
      end
      Rails.logger.warn("#{@opts[:uuid]}/#{@opts[:file]}: #{$?}") if $? != 0
    end
  end
end
