class CollectionsController < ApplicationController
  skip_before_filter :find_object_by_uuid, :only => [:graph]
  skip_before_filter :check_user_agreements, :only => [:show_file]

  def graph
    index
  end

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

  def generate_edges(gr, uuid, edge_added = false)
    m = /([a-f0-9]{32}(\+[0-9]+)?)(\+.*)?/.match(uuid)
    if m  
      # uuid is a collection
      uuid = m[1]
      gr += "\"#{uuid}\" [href=\"#{collection_path uuid}\"];"

      Job.where(output: uuid).each do |job|
        gr += "\"#{job.uuid}\" [href=\"#{job_path job.uuid}\"];"
        gr += "\"#{uuid}\" -> \"#{job.uuid}\" [label=\" output\"];"
        gr = generate_edges(gr, job.uuid)
      end

      Job.where(log: uuid).each do |job|
        gr += "\"#{job.uuid}\" [href=\"#{job_path job.uuid}\"];"
        gr += "\"#{uuid}\" -> \"#{job.uuid}\" [label=\" log\"];"
        gr = generate_edges(gr, job.uuid)
      end
      
    else
      # uuid is something else
      rsc = ArvadosBase::resource_class_for_uuid uuid
      gr += "\"#{uuid}\" [href=\"#{rsc}/#{uuid}\"];"
      
      if rsc.to_s == "Job"
        Job.where(uuid: uuid).each do |job|
          job.script_parameters.each do |k, v|
            gr += "\"#{job.uuid}\" [href=\"#{job_path job.uuid}\"];"
            gr += "\"#{job.uuid}\" -> \"#{v}\" [label=\" #{k}\"];"
            gr = generate_edges(gr, v)
          end
        end
      end
     
      gr
    end

    Link.where(head_uuid: uuid, link_class: "provenance", name: "provided").each do |link|
      rsc = ArvadosBase::resource_class_for_uuid link.tail_uuid
      puts "rsc is #{rsc}"
      gr += "\"#{link.tail_uuid}\" [href=\"#{rsc}/#{link.tail_uuid}\"];"
      gr += "\"#{link.head_uuid}\" -> \"#{link.tail_uuid}\" [label=\" provided\"];"
      generate_edges(gr, link.tail_uuid)
    end

    gr
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

    require 'open3'
    
    gr = "digraph {"
    gr += "node [fontsize=8];"
    gr += "edge [dir=back,fontsize=8];"
    
    gr = generate_edges(gr, @object.uuid)

    gr += "}"
    @prov_svg = ""

    Open3.popen2("dot", "-Tsvg") do |stdin, stdout, wait_thr|
      stdin.print(gr)
      stdin.close
      wait_thr.value
      @prov_svg = stdout.read()
      stdout.close()
    end

    @prov_svg = @prov_svg.sub(/<\?xml.*?\?>/m, "")
    @prov_svg = @prov_svg.sub(/<!DOCTYPE.*?>/m, "")
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
