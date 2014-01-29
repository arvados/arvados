class CollectionsController < ApplicationController
  skip_before_filter :find_object_by_uuid, :only => [:provenance]
  skip_before_filter :check_user_agreements, :only => [:show_file]

  def index
    if params[:search].andand.length.andand > 0
      tags = Link.where(any: ['contains', params[:search]])
      @collections = (Collection.where(uuid: tags.collect(&:head_uuid)) |
                      Collection.where(any: ['contains', params[:search]])).
        uniq { |c| c.uuid }
    else
      @collections = Collection.limit(100)
    end
    @links = Link.limit(1000).
      where(head_uuid: @collections.collect(&:uuid))
    @collection_info = {}
    @collections.each do |c|
      @collection_info[c.uuid] = {
        tags: [],
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
        info[:tags] << link.name
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
    opts = params.merge(arvados_api_token: Thread.current[:arvados_api_token])
    if r = params[:file].match(/(\.\w+)/)
      ext = r[1]
    end
    self.response.headers['Content-Type'] =
      Rack::Mime::MIME_TYPES[ext] || 'application/octet-stream'
    self.response.headers['Content-Length'] = params[:size] if params[:size]
    self.response.headers['Content-Disposition'] = params[:disposition] if params[:disposition]
    self.response_body = FileStreamer.new opts
  end

  def self.describe_node(uuid)
    uuid = uuid.to_s
    rsc = ArvadosBase::resource_class_for_uuid uuid
    if rsc
      "\"#{uuid}\" [label=\"#{rsc}\\n#{uuid}\",href=\"/#{rsc.to_s.underscore.pluralize rsc}/#{uuid}\"];"
    else
      ""
    end
  end

  def self.job_uuid(job)
    # "#{job[:script]}\\n#{job[:script_version]}"
    "#{job[:script]}"
  end

  def self.collection_uuid(uuid)
    m = /([a-f0-9]{32}(\+[0-9]+)?)(\+.*)?/.match(uuid.to_s)
    if m
      m[1]
    else
      nil
    end
  end

  def self.script_param_edges(visited, job, prefix, sp, opts)
    gr = ""
    if sp and not sp.empty?
      case sp
      when Hash
        sp.each do |k, v|
          if prefix.size > 0
            k = prefix + "::" + k.to_s
          end
          gr += CollectionsController::script_param_edges(visited, job, k.to_s, v, opts)
        end
      when Array
        sp.each do |v|
          gr += CollectionsController::script_param_edges(visited, job, prefix, v, opts)
        end
      else
        m = collection_uuid(sp)
        if m
          gr += "\"#{job_uuid(job)}\" -> \"#{m}\" [label=\" #{prefix}\"];"
          gr += CollectionsController::generate_provenance_edges(visited, m, opts)
        end
      end
    end
    gr
  end

  def self.generate_provenance_edges(pdata, uuid, opts)
    gr = ""
    m = CollectionsController::collection_uuid(uuid)
    uuid = m if m

    uuid = uuid.intern if uuid

    if (not uuid) or uuid.empty? \
      or (pdata[uuid] and pdata[uuid][:_visited])

      #puts "already visited #{uuid}"
      return ""
    end

    if not pdata[uuid] then 
      return CollectionsController::describe_node(uuid) 
    else
      pdata[uuid][:_visited] = true
    end

    #puts "visiting #{uuid}"

    if m  
      # uuid is a collection
      gr += CollectionsController::describe_node(uuid)

      pdata.each do |k, job|
        if job[:output] == uuid.to_s
          gr += "\"#{uuid}\" -> \"#{job_uuid(job)}\" [label=\"output\"];"
          gr += CollectionsController::generate_provenance_edges(pdata, job[:uuid])
        end
        if job[:log] == uuid.to_s
          gr += "\"#{uuid}\" -> \"#{job_uuid(job)}\" [label=\"log\"];"
          gr += CollectionsController::generate_provenance_edges(pdata, job[:uuid])
        end
      end
    else
      # uuid is something else
      rsc = ArvadosBase::resource_class_for_uuid uuid.to_s

      if rsc == Job
        job = pdata[uuid]
        if job
          gr += CollectionsController::script_param_edges(pdata, job, "", job[:script_parameters], opts)
        end
      else
        gr += CollectionsController::describe_node(uuid)
      end
    end

    pdata.each do |k, link|
      if link[:head_uuid] == uuid.to_s and link[:link_class] == "provenance"
        gr += CollectionsController::describe_node(link[:tail_uuid])
        gr += "\"#{link[:head_uuid]}\" -> \"#{link[:tail_uuid]}\" [label=\" #{link[:name]}\", href=\"/links/#{link[:uuid]}\"];"
        gr += CollectionsController::generate_provenance_edges(pdata, link[:tail_uuid], opts)
      end
    end

    #puts "finished #{uuid}"

    gr
  end

  def self.create_provenance_graph(pdata, uuid, opts={})
    require 'open3'
    
    gr = """strict digraph {
node [fontsize=8,shape=box];
edge [dir=back,fontsize=8];"""

    #puts "pdata is #{pdata}"

    gr += CollectionsController::generate_provenance_edges(pdata, uuid, opts)

    gr += "}"
    svg = ""

    Open3.popen2("dot", "-Tsvg") do |stdin, stdout, wait_thr|
      stdin.print(gr)
      stdin.close
      svg = stdout.read()
      wait_thr.value
      stdout.close()
    end

    svg = svg.sub(/<\?xml.*?\?>/m, "")
    svg = svg.sub(/<!DOCTYPE.*?>/m, "")
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
      end
    end
    Link.where(tail_uuid: @sourcedata.keys).each do |link|
      if link.link_class == 'data_origin'
        @sourcedata[link.tail_uuid][:data_origins] ||= []
        @sourcedata[link.tail_uuid][:data_origins] << [link.name, link.head_kind, link.head_uuid]
      end
    end
    Collection.where(uuid: @sourcedata.keys).each do |collection|
      if @sourcedata[collection.uuid]
        @sourcedata[collection.uuid][:collection] = collection
      end
    end
    
    Collection.where(uuid: @object.uuid).each do |u|
      @prov_svg = CollectionsController::create_provenance_graph u.provenance, u.uuid
    end
  end

  protected
  class FileStreamer
    def initialize(opts={})
      @opts = opts
    end
    def each
      return unless @opts[:uuid] && @opts[:file]
      env = Hash[ENV].
        merge({
                'ARVADOS_API_HOST' =>
                $arvados_api_client.arvados_v1_base.
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
