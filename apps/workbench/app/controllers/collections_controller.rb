class CollectionsController < ApplicationController
  skip_around_filter :thread_with_mandatory_api_token, only: [:show_file]
  skip_before_filter :find_object_by_uuid, only: [:provenance, :show_file]
  skip_before_filter :check_user_agreements, only: [:show_file]

  def show_pane_list
    %w(Files Attributes Metadata Provenance_graph Used_by JSON API)
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

  def index
    if params[:search].andand.length.andand > 0
      tags = Link.where(any: ['contains', params[:search]])
      @collections = (Collection.where(uuid: tags.collect(&:head_uuid)) |
                      Collection.where(any: ['contains', params[:search]])).
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

      @collections = Collection.limit(limit).offset(offset)
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
  end

  def show_file
    # We pipe from arv-get to send the file to the user.  Before we start it,
    # we ask the API server if the file actually exists.  This serves two
    # purposes: it lets us return a useful status code for common errors, and
    # helps us figure out which token to provide to arv-get.
    coll = nil
    usable_token = find_usable_token do
      coll = Collection.find(params[:uuid])
    end
    if usable_token.nil?
      return  # Response already rendered.
    elsif params[:file].nil? or not file_in_collection?(coll, params[:file])
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

  def show
    return super if !@object
    @provenance = []
    @output2job = {}
    @output2colorindex = {}
    @sourcedata = {params[:uuid] => {uuid: params[:uuid]}}
    @protected = {}

    colorindex = -1
    any_hope_left = true
    while any_hope_left
      any_hope_left = false
      Job.where(output: @sourcedata.keys).sort_by { |a| a.finished_at || a.created_at }.reverse.each do |job|
        if !@output2colorindex[job.output]
          any_hope_left = true
          @output2colorindex[job.output] = (colorindex += 1) % 10
          @provenance << {job: job, output: job.output}
          @sourcedata.delete job.output
          @output2job[job.output] = job
          job.dependencies.each do |new_source_data|
            unless @output2colorindex[new_source_data]
              @sourcedata[new_source_data] = {uuid: new_source_data}
            end
          end
        end
      end
    end

    Link.where(head_uuid: @sourcedata.keys | @output2job.keys).each do |link|
      if link.link_class == 'resources' and link.name == 'wants'
        @protected[link.head_uuid] = true
        if link.tail_uuid == current_user.uuid
          @is_persistent = true
        end
      end
    end
    Link.where(tail_uuid: @sourcedata.keys).each do |link|
      if link.link_class == 'data_origin'
        @sourcedata[link.tail_uuid][:data_origins] ||= []
        @sourcedata[link.tail_uuid][:data_origins] << [link.name, link.head_uuid]
      end
    end
    Collection.where(uuid: @sourcedata.keys).each do |collection|
      if @sourcedata[collection.uuid]
        @sourcedata[collection.uuid][:collection] = collection
      end
    end

    Collection.where(uuid: @object.uuid).each do |u|
      @prov_svg = ProvenanceHelper::create_provenance_graph(u.provenance, "provenance_svg",
                                                            {:request => request,
                                                              :direction => :bottom_up,
                                                              :combine_jobs => :script_only}) rescue nil
      @used_by_svg = ProvenanceHelper::create_provenance_graph(u.used_by, "used_by_svg",
                                                               {:request => request,
                                                                 :direction => :top_down,
                                                                 :combine_jobs => :script_only,
                                                                 :pdata_only => true}) rescue nil
    end
  end

  protected

  def find_usable_token
    # Iterate over every token available to make it the current token and
    # yield the given block.
    # If the block succeeds, return the token it used.
    # Otherwise, render an error response based on the most specific
    # error we encounter, and return nil.
    read_tokens = [Thread.current[:arvados_api_token]].compact
    if params[:reader_tokens].is_a? Array
      read_tokens += params[:reader_tokens]
    end
    most_specific_error = [401]
    read_tokens.each do |api_token|
      using_specific_api_token(api_token) do
        begin
          yield
          return api_token
        rescue ArvadosApiClient::NotLoggedInException => error
          status = 401
        rescue => error
          status = (error.message =~ /\[API: (\d+)\]$/) ? $1.to_i : nil
          raise unless [401, 403, 404].include?(status)
        end
        if status >= most_specific_error.first
          most_specific_error = [status, error]
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

  def file_in_collection?(collection, filename)
    def normalized_path(part_list)
      File.join(part_list).sub(%r{^\./}, '')
    end
    target = normalized_path([filename])
    collection.files.each do |file_spec|
      return true if (normalized_path(file_spec[0, 2]) == target)
    end
    false
  end

  def file_enumerator(opts)
    FileStreamer.new opts
  end

  class FileStreamer
    def initialize(opts={})
      @opts = opts
    end
    def each
      return unless @opts[:uuid] && @opts[:file]
      env = Hash[ENV].
        merge({
                'ARVADOS_API_HOST' =>
                arvados_api_client.arvados_v1_base.
                sub(/\/arvados\/v1/, '').
                sub(/^https?:\/\//, ''),
                'ARVADOS_API_TOKEN' =>
                @opts[:arvados_api_token],
                'ARVADOS_API_HOST_INSECURE' =>
                Rails.configuration.arvados_insecure_https ? 'true' : 'false'
              })
      IO.popen([env, 'arv-get', "#{@opts[:uuid]}/#{@opts[:file]}"],
               'rb') do |io|
        while buf = io.read(2**20)
          yield buf
        end
      end
      Rails.logger.warn("#{@opts[:uuid]}/#{@opts[:file]}: #{$?}") if $? != 0
    end
  end
end
