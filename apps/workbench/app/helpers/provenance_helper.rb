module ProvenanceHelper

  class GenerateGraph
    def initialize(pdata, opts)
      @pdata = pdata
      @opts = opts
      @visited = {}
      @jobs = {}
      @node_extra = {}
    end

    def self.collection_uuid(uuid)
      Keep::Locator.parse(uuid).andand.strip_hints.andand.to_s
    end

    def url_for u
      p = { :host => @opts[:request].host,
        :port => @opts[:request].port,
        :protocol => @opts[:request].protocol }
      p.merge! u
      Rails.application.routes.url_helpers.url_for (p)
    end

    def determine_fillcolor(n)
      fillcolor = %w(666666 669966 666699 666666 996666)[n || 0] || '666666'
      "style=\"filled\",color=\"#ffffff\",fillcolor=\"##{fillcolor}\",fontcolor=\"#ffffff\""
    end

    def describe_node(uuid, describe_opts={})
      bgcolor = determine_fillcolor (describe_opts[:pip] || @opts[:pips].andand[uuid])

      rsc = ArvadosBase::resource_class_for_uuid uuid

      if GenerateGraph::collection_uuid(uuid) || rsc == Collection
        if Collection.is_empty_blob_locator? uuid.to_s
          # special case
          return "\"#{uuid}\" [label=\"(empty collection)\"];\n"
        end

        href = url_for ({:controller => Collection.to_s.tableize,
                          :action => :show,
                          :id => uuid.to_s })

        return "\"#{uuid}\" [label=\"#{encode_quotes(describe_opts[:label] || (@pdata[uuid] and @pdata[uuid][:name]) || uuid)}\",shape=box,href=\"#{href}\",#{bgcolor}];\n"
      else
        href = ""
        if describe_opts[:href]
          href = ",href=\"#{url_for ({:controller => describe_opts[:href][:controller],
                            :action => :show,
                            :id => describe_opts[:href][:id] })}\""
        end
        return "\"#{uuid}\" [label=\"#{encode_quotes(describe_opts[:label] || uuid)}\",#{bgcolor},shape=#{describe_opts[:shape] || 'box'}#{href}];\n"
      end
    end

    def job_uuid(job)
      d = Digest::MD5.hexdigest(job[:script_parameters].to_json)
      if @opts[:combine_jobs] == :script_only
        uuid = "#{job[:script]}_#{d}"
      elsif @opts[:combine_jobs] == :script_and_version
        uuid = "#{job[:script]}_#{job[:script_version]}_#{d}"
      else
        uuid = "#{job[:uuid]}"
      end

      @jobs[uuid] = [] unless @jobs[uuid]
      @jobs[uuid] << job unless @jobs[uuid].include? job

      uuid
    end

    def edge(tail, head, extra)
      if @opts[:direction] == :bottom_up
        gr = "\"#{encode_quotes head}\" -> \"#{encode_quotes tail}\""
      else
        gr = "\"#{encode_quotes tail}\" -> \"#{encode_quotes head}\""
      end

      if extra.length > 0
        gr += " ["
        extra.each do |k, v|
          gr += "#{k}=\"#{encode_quotes v}\","
        end
        gr += "]"
      end
      gr += ";\n"
      gr
    end

    def script_param_edges(uuid, sp)
      gr = ""

      sp.each do |k, v|
        if @opts[:all_script_parameters]
          if v.is_a? Array or v.is_a? Hash
            encv = JSON.pretty_generate(v).gsub("\n", "\\l") + "\\l"
          else
            encv = v.to_json
          end
          gr += "\"#{encode_quotes encv}\" [shape=box];\n"
          gr += edge(encv, uuid, {:label => k})
        end
      end
      gr
    end

    def job_edges job, edge_opts={}
      uuid = job_uuid(job)
      gr = ""

      ProvenanceHelper::find_collections job[:script_parameters] do |collection_hash, collection_uuid, key|
        if collection_uuid
          gr += describe_node(collection_uuid)
          gr += edge(collection_uuid, uuid, {:label => key})
        else
          gr += describe_node(collection_hash)
          gr += edge(collection_hash, uuid, {:label => key})
        end
      end

      if job[:docker_image_locator] and !@opts[:no_docker]
        gr += describe_node(job[:docker_image_locator], {label: (job[:runtime_constraints].andand[:docker_image] || job[:docker_image_locator])})
        gr += edge(job[:docker_image_locator], uuid, {label: "docker_image"})
      end

      if @opts[:script_version_nodes]
        gr += describe_node(job[:script_version], {:label => "git:#{job[:script_version]}"})
        gr += edge(job[:script_version], uuid, {:label => "script_version"})
      end

      if job[:output] and !edge_opts[:no_output]
        gr += describe_node(job[:output])
        gr += edge(uuid, job[:output], {label: "output" })
      end

      if job[:log] and !edge_opts[:no_log]
        gr += describe_node(job[:log])
        gr += edge(uuid, job[:log], {label: "log"})
      end

      gr
    end

    def generate_provenance_edges(uuid)
      gr = ""
      m = GenerateGraph::collection_uuid(uuid)
      uuid = m if m

      if uuid.nil? or uuid.empty? or @visited[uuid]
        return ""
      end

      if @pdata[uuid].nil?
        return ""
      else
        @visited[uuid] = true
      end

      if uuid.start_with? "component_"
        # Pipeline component inputs
        job = @pdata[@pdata[uuid][:job].andand[:uuid]]

        if job
          gr += describe_node(job_uuid(job), {label: uuid[38..-1], pip: @opts[:pips].andand[job[:uuid]], shape: "oval",
                                href: {controller: 'jobs', id: job[:uuid]}})
          gr += job_edges job, {no_output: true, no_log: true}
        end

        # Pipeline component output
        outuuid = @pdata[uuid][:output_uuid]
        if outuuid
          outcollection = @pdata[outuuid]
          if outcollection
            gr += edge(job_uuid(job), outcollection[:portable_data_hash], {label: "output"})
            gr += describe_node(outcollection[:portable_data_hash], {label: outcollection[:name]})
          end
        elsif job and job[:output]
          gr += describe_node(job[:output])
          gr += edge(job_uuid(job), job[:output], {label: "output" })
        end
      else
        rsc = ArvadosBase::resource_class_for_uuid uuid

        if rsc == Job
          job = @pdata[uuid]
          gr += job_edges job if job
        end
      end

      @pdata.each do |k, link|
        if link[:head_uuid] == uuid.to_s and link[:link_class] == "provenance"
          href = url_for ({:controller => Link.to_s.tableize,
                            :action => :show,
                            :id => link[:uuid] })

          gr += describe_node(link[:tail_uuid])
          gr += edge(link[:head_uuid], link[:tail_uuid], {:label => link[:name], :href => href})
          gr += generate_provenance_edges(link[:tail_uuid])
        end
      end

      gr
    end

    def describe_jobs
      gr = ""
      @jobs.each do |k, v|
        href = url_for ({:controller => Job.to_s.tableize,
                          :action => :index })

        gr += "\"#{k}\" [href=\"#{href}?"

        n = 0
        v.each do |u|
          gr += ";" unless gr.end_with? "?"
          gr += "uuid%5b%5d=#{u[:uuid]}"
          n |= @opts[:pips][u[:uuid]] if @opts[:pips] and @opts[:pips][u[:uuid]]
        end

        gr += "\",label=\""

        label = "#{v[0][:script]}"

        if label == "run-command" and v[0][:script_parameters][:command].is_a? Array
          label = v[0][:script_parameters][:command].join(' ')
        end

        if not @opts[:combine_jobs]
          label += "\\n#{v[0][:finished_at]}"
        end

        gr += encode_quotes label

        gr += "\",#{determine_fillcolor n}];\n"
      end
      gr
    end

    def encode_quotes value
      value.to_s.gsub("\"", "\\\"").gsub("\n", "\\n")
    end
  end

  def self.create_provenance_graph(pdata, svgId, opts={})
    if pdata.is_a? Array or pdata.is_a? ArvadosResourceList
      p2 = {}
      pdata.each do |k|
        p2[k[:uuid]] = k if k[:uuid]
      end
      pdata = p2
    end

    unless pdata.is_a? Hash
      raise "create_provenance_graph accepts Array or Hash for pdata only, pdata is #{pdata.class}"
    end

    gr = """strict digraph {
node [fontsize=10,fontname=\"Helvetica,Arial,sans-serif\"];
edge [fontsize=10,fontname=\"Helvetica,Arial,sans-serif\"];
"""

    if opts[:direction] == :bottom_up
      gr += "edge [dir=back];"
    end

    begin
      pdata = pdata.stringify_keys

      g = GenerateGraph.new(pdata, opts)

      pdata.each do |k, v|
        if !opts[:only_components] or k.start_with? "component_"
          gr += g.generate_provenance_edges(k)
        else
          #gr += describe_node(k)
        end
      end

      if !opts[:only_components]
        gr += g.describe_jobs
      end

    rescue => e
      Rails.logger.warn "#{e.inspect}"
      Rails.logger.warn "#{e.backtrace.join("\n\t")}"
      raise
    end

    gr += "}"
    svg = ""

    require 'open3'

    Open3.popen2("dot", "-Tsvg") do |stdin, stdout, wait_thr|
      stdin.print(gr)
      stdin.close
      svg = stdout.read()
      wait_thr.value
      stdout.close()
    end

    svg = svg.sub(/<\?xml.*?\?>/m, "")
    svg = svg.sub(/<!DOCTYPE.*?>/m, "")
    svg = svg.sub(/<svg /, "<svg id=\"#{svgId}\" ")
  end

  # yields hash, uuid
  # Position indicates whether it is a content hash or arvados uuid.
  # One will hold a value, the other will always be nil.
  def self.find_collections(sp, key=nil, &b)
    case sp
    when ArvadosBase
      sp.class.columns.each do |c|
        find_collections(sp[c.name.to_sym], nil, &b)
      end
    when Hash
      sp.each do |k, v|
        find_collections(v, key || k, &b)
      end
    when Array
      sp.each do |v|
        find_collections(v, key, &b)
      end
    when String
      if m = /[a-f0-9]{32}\+\d+/.match(sp)
        yield m[0], nil, key
      elsif m = /[0-9a-z]{5}-4zz18-[0-9a-z]{15}/.match(sp)
        yield nil, m[0], key
      end
    end
  end
end
