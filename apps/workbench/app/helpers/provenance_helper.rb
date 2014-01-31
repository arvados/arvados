module ProvenanceHelper

  class GenerateGraph
    def initialize(pdata, opts)
      @pdata = pdata
      @opts = opts
      @visited = {}
      @jobs = {}
    end

    def self.collection_uuid(uuid)
      m = /^([a-f0-9]{32}(\+[0-9]+)?)(\+.*)?$/.match(uuid.to_s)
      if m
        #if m[2]
        return m[1]
        #else
        #  Collection.where(uuid: ['contains', m[1]]).each do |u|
        #    puts "fixup #{uuid} to #{u.uuid}"
        #    return u.uuid
        #  end
        #end
      else
        nil
      end
    end

    def determine_fillcolor(n)
      bgcolor = ""
      case n
      when 1
        bgcolor = "style=filled,fillcolor=\"#88ff88\""
      when 2
        bgcolor = "style=filled,fillcolor=\"#8888ff\""
      when 3
        bgcolor = "style=filled,fillcolor=\"#88ffff\""
      end
      bgcolor
    end

    def describe_node(uuid)
      bgcolor = determine_fillcolor @opts[:pips][uuid] if @opts[:pips]

      rsc = ArvadosBase::resource_class_for_uuid uuid.to_s
      if rsc
        href = "/#{rsc.to_s.underscore.pluralize rsc}/#{uuid}"
      
        #"\"#{uuid}\" [label=\"#{rsc}\\n#{uuid}\",href=\"#{href}\"];\n"
        if rsc == Collection
          if @pdata[uuid] 
            #puts @pdata[uuid]
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
                return "\"#{uuid}\" [label=\"#{label}\",href=\"#{href}\",shape=oval,#{bgcolor}];\n"
              end
            end  
          end
          return "\"#{uuid}\" [label=\"#{rsc}\",href=\"#{href}\",#{bgcolor}];\n"
        end
      end
      "\"#{uuid}\" [#{bgcolor}];\n"
    end

    def job_uuid(job)
      if @opts[:combine_jobs] == :script_only
        uuid = "#{job[:script]}"
      elsif @opts[:combine_jobs] == :script_and_version
        uuid = "#{job[:script]}_#{job[:script_version]}"
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
        gr += "["
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
      if sp and not sp.empty?
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
          sp.each do |v|
            if GenerateGraph::collection_uuid(v)
              gr += script_param_edges(job, "#{prefix}[#{i}]", v)
            else
              node += "', '" unless node == ""
              node = "['" if node == ""
              node += "#{v}"
            end
            i += 1
          end
          unless node == ""
            node += "']"
            #puts node
            #id = "#{job[:uuid]}_#{prefix}"
            gr += "\"#{node}\" [label=\"#{node}\"];\n"
            gr += edge(job_uuid(job), node, {:label => prefix})        
          end
        else
          m = GenerateGraph::collection_uuid(sp)
          if m
            gr += edge(job_uuid(job), m, {:label => prefix})
            gr += generate_provenance_edges(m)
          elsif @opts[:all_script_parameters]
            #id = "#{job[:uuid]}_#{prefix}"
            gr += "\"#{sp}\" [label=\"#{sp}\"];\n"
            gr += edge(job_uuid(job), sp, {:label => prefix})
          end
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

        #puts "already @visited #{uuid}"
        return ""
      end

      if not @pdata[uuid] then 
        return describe_node(uuid)
      else
        @visited[uuid] = true
      end

      #puts "visiting #{uuid}"

      if m  
        # uuid is a collection
        gr += describe_node(uuid)

        @pdata.each do |k, job|
          if job[:output] == uuid.to_s
            gr += edge(uuid, job_uuid(job), {:label => "output"})
            gr += generate_provenance_edges(job[:uuid])
          end
          if job[:log] == uuid.to_s
            gr += edge(uuid, job_uuid(job), {:label => "log"})
            gr += generate_provenance_edges(job[:uuid])
          end
        end
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
        else
          gr += describe_node(uuid)
        end
      end

      @pdata.each do |k, link|
        if link[:head_uuid] == uuid.to_s and link[:link_class] == "provenance"
          gr += describe_node(link[:tail_uuid])
          gr += edge(link[:head_uuid], link[:tail_uuid], {:label => link[:name], :href => "/links/#{link[:uuid]}"}) 
          gr += generate_provenance_edges(link[:tail_uuid])
        end
      end

      #puts "finished #{uuid}"

      gr
    end

    def describe_jobs
      gr = ""
      @jobs.each do |k, v|
        gr += "\"#{k}\" [href=\"/jobs?"
        
        n = 0
        v.each do |u|
          gr += "uuid%5b%5d=#{u[:uuid]}&"
          n |= @opts[:pips][u[:uuid].intern] if @opts[:pips] and @opts[:pips][u[:uuid].intern]
        end

        gr += "\",label=\""
        
        if @opts[:combine_jobs] == :script_only
          gr += uuid = "#{v[0][:script]}"
        elsif @opts[:combine_jobs] == :script_and_version
          gr += uuid = "#{v[0][:script]}"
        else
          gr += uuid = "#{v[0][:script]}\\n#{v[0][:finished_at]}"
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
node [fontsize=8,shape=box];
edge [fontsize=8];
"""

    if opts[:direction] == :bottom_up
      gr += "edge [dir=back];"
    end

    #puts "@pdata is #{pdata}"

    g = GenerateGraph.new(pdata, opts)

    pdata.each do |k, v|
      gr += g.generate_provenance_edges(k)
    end

    gr += g.describe_jobs

    gr += "}"
    svg = ""

    #puts gr

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
    if sp and not sp.empty?
      case sp
      when Hash
        sp.each do |k, v|
          c.concat(find_collections(v))
        end
      when Array
        sp.each do |v|
          c.concat(find_collections(v))
        end
      else
        m = GenerateGraph::collection_uuid(sp)
        if m
          c << m
        end
      end
    end
    c
  end
end
