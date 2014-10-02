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
      m = CollectionsHelper.match(uuid)
      if m
        if m[2]
          return m[1]+m[2]
        else
          return m[1]
        end
      else
        nil
      end
    end

    def url_for u
      p = { :host => @opts[:request].host,
        :port => @opts[:request].port,
        :protocol => @opts[:request].protocol }
      p.merge! u
      Rails.application.routes.url_helpers.url_for (p)
    end

    def determine_fillcolor(n)
      fillcolor = %w(aaaaaa aaffaa aaaaff aaaaaa ffaaaa)[n || 0] || 'aaaaaa'
      "style=filled,fillcolor=\"##{fillcolor}\""
    end

    def describe_node(uuid)
      uuid = uuid.to_sym
      bgcolor = determine_fillcolor @opts[:pips].andand[uuid]

      rsc = ArvadosBase::resource_class_for_uuid uuid.to_s
      if rsc
        href = url_for ({:controller => rsc.to_s.tableize,
                          :action => :show,
                          :id => uuid.to_s })

        #"\"#{uuid}\" [label=\"#{rsc}\\n#{uuid}\",href=\"#{href}\"];\n"
        if rsc == Collection
          if Collection.is_empty_blob_locator? uuid.to_s
            # special case
            return "\"#{uuid}\" [label=\"(empty collection)\"];\n"
          end
          if @pdata[uuid]
            if @pdata[uuid][:name]
              return "\"#{uuid}\" [label=\"#{@pdata[uuid][:name]}\",href=\"#{href}\",shape=oval,#{bgcolor}];\n"
            else
              files = nil
              if @pdata[uuid].respond_to? :files
                files = @pdata[uuid].files
              elsif @pdata[uuid][:files]
                files = @pdata[uuid][:files]
              end

              if files
                i = 0
                label = ""
                while i < 3 and i < files.length
                  label += "\\n" unless label == ""
                  label += files[i][1]
                  i += 1
                end
                if i < files.length
                  label += "\\n&vellip;"
                end
                extra_s = @node_extra[uuid].andand.map { |k,v|
                  "#{k}=\"#{v}\""
                }.andand.join ","
                return "\"#{uuid}\" [label=\"#{label}\",href=\"#{href}\",shape=oval,#{bgcolor},#{extra_s}];\n"
              end
            end
          end
        end
        return "\"#{uuid}\" [label=\"#{rsc}\",href=\"#{href}\",#{bgcolor}];\n"
      end
      "\"#{uuid}\" [#{bgcolor}];\n"
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
        gr = "\"#{tail}\" -> \"#{head}\""
      else
        gr = "\"#{head}\" -> \"#{tail}\""
      end
      if extra.length > 0
        gr += " ["
        extra.each do |k, v|
          gr += "#{k}=\"#{v}\","
        end
        gr += "]"
      end
      gr += ";\n"
      gr
    end

    def script_param_edges(job, prefix, sp)
      gr = ""
      case sp
      when Hash
        sp.each do |k, v|
          if prefix.size > 0
            k = prefix + "::" + k.to_s
          end
          gr += script_param_edges(job, k.to_s, v)
        end
      when Array
        i = 0
        node = ""
        count = 0
        sp.each do |v|
          if GenerateGraph::collection_uuid(v)
            gr += script_param_edges(job, "#{prefix}[#{i}]", v)
          elsif @opts[:all_script_parameters]
            t = "#{v}"
            nl = (if (count+t.length) > 60 then "\\n" else " " end)
            count = 0 if (count+t.length) > 60
            node += "',#{nl}'" unless node == ""
            node = "['" if node == ""
            node += t
            count += t.length
          end
          i += 1
        end
        unless node == ""
          node += "']"
          node_value = "#{node}".gsub("\"", "\\\"")
          gr += "\"#{node_value}\" [label=\"#{node_value}\"];\n"
          gr += edge(job_uuid(job), node_value, {:label => prefix})
        end
      when String
        return '' if sp.empty?
        m = GenerateGraph::collection_uuid(sp)
        if m and (@pdata[m.intern] or (not @opts[:pdata_only]))
          gr += edge(job_uuid(job), m, {:label => prefix})
          gr += generate_provenance_edges(m)
        elsif @opts[:all_script_parameters]
          gr += "\"#{sp}\" [label=\"#{sp}\"];\n"
          gr += edge(job_uuid(job), sp, {:label => prefix})
        end
      end
      gr
    end

    def generate_provenance_edges(uuid)
      gr = ""
      m = GenerateGraph::collection_uuid(uuid)
      uuid = m if m

      uuid = uuid.intern if uuid

      if (not uuid) or uuid.empty? or @visited[uuid]
        return ""
      end

      if not @pdata[uuid] then
        return describe_node(uuid)
      else
        @visited[uuid] = true
      end

      if m
        # uuid is a collection
        if not Collection.is_empty_blob_locator? uuid.to_s
          @pdata.each do |k, job|
            if job[:output] == uuid.to_s
              extra = { label: 'output' }
              gr += edge(uuid, job_uuid(job), extra)
              gr += generate_provenance_edges(job[:uuid])
            end
            if job[:log] == uuid.to_s
              gr += edge(uuid, job_uuid(job), {:label => "log"})
              gr += generate_provenance_edges(job[:uuid])
            end
          end
        end
        gr += describe_node(uuid)
      else
        # uuid is something else
        rsc = ArvadosBase::resource_class_for_uuid uuid.to_s

        if rsc == Job
          job = @pdata[uuid]
          if job
            gr += script_param_edges(job, "", job[:script_parameters])

            if @opts[:script_version_nodes]
              gr += describe_node(job[:script_version])
              gr += edge(job_uuid(job), job[:script_version], {:label => "script_version"})
            end
          end
        elsif rsc == Link
          # do nothing
        else
          gr += describe_node(uuid)
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
          gr += "uuid%5b%5d=#{u[:uuid]}&"
          n |= @opts[:pips][u[:uuid].intern] if @opts[:pips] and @opts[:pips][u[:uuid].intern]
        end

        gr += "\",label=\""

        if @opts[:combine_jobs] == :script_only
          gr += "#{v[0][:script]}"
        elsif @opts[:combine_jobs] == :script_and_version
          gr += "#{v[0][:script]}" # Just show the name but the nodes will be distinct
        else
          gr += "#{v[0][:script]}\\n#{v[0][:finished_at]}"
        end
        gr += "\",#{determine_fillcolor n}];\n"
      end
      gr
    end

  end

  def self.create_provenance_graph(pdata, svgId, opts={})
    if pdata.is_a? Array or pdata.is_a? ArvadosResourceList
      p2 = {}
      pdata.each do |k|
        p2[k[:uuid].intern] = k if k[:uuid]
      end
      pdata = p2
    end

    unless pdata.is_a? Hash
      raise "create_provenance_graph accepts Array or Hash for pdata only, pdata is #{pdata.class}"
    end

    gr = """strict digraph {
node [fontsize=10,shape=box];
edge [fontsize=10];
"""

    if opts[:direction] == :bottom_up
      gr += "edge [dir=back];"
    end

    g = GenerateGraph.new(pdata, opts)

    pdata.each do |k, v|
      gr += g.generate_provenance_edges(k)
    end

    gr += g.describe_jobs

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

  def self.find_collections(sp)
    c = []
    case sp
    when Hash
      sp.each do |k, v|
        c.concat(find_collections(v))
      end
    when Array
      sp.each do |v|
        c.concat(find_collections(v))
      end
    when String
      if !sp.empty?
        m = GenerateGraph::collection_uuid(sp)
        if m
          c << m
        end
      end
    end
    c
  end
end
