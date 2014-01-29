module ProvenanceHelper
  def self.describe_node(pdata, uuid)
    rsc = ArvadosBase::resource_class_for_uuid uuid.to_s
    if rsc
      href = "/#{rsc.to_s.underscore.pluralize rsc}/#{uuid}"

      #"\"#{uuid}\" [label=\"#{rsc}\\n#{uuid}\",href=\"#{href}\"];\n"
      if rsc == Collection
        if pdata[uuid] 
          #puts pdata[uuid]
          if pdata[uuid][:name]
            return "\"#{uuid}\" [label=\"#{pdata[uuid][:name]}\",href=\"#{href}\",shape=oval];\n"
          else
            files = nil
            if pdata[uuid].respond_to? :files
              files = pdata[uuid].files
            elsif pdata[uuid][:files]
              files = pdata[uuid][:files]
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
              return "\"#{uuid}\" [label=\"#{label}\",href=\"#{href}\",shape=oval];\n"
            end
          end  
        end
        return "\"#{uuid}\" [label=\"#{rsc}\",href=\"#{href}\"];\n"
      end
    end
    return ""
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

  def self.edge(tail, head, extra, opts)
    if opts[:direction] == :bottom_up
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

  def self.script_param_edges(pdata, visited, job, prefix, sp, opts)
    gr = ""
    if sp and not sp.empty?
      case sp
      when Hash
        sp.each do |k, v|
          if prefix.size > 0
            k = prefix + "::" + k.to_s
          end
          gr += ProvenanceHelper::script_param_edges(pdata, visited, job, k.to_s, v, opts)
        end
      when Array
        i = 0
        node = ""
        sp.each do |v|
          if collection_uuid(v)
            gr += ProvenanceHelper::script_param_edges(pdata, visited, job, "#{prefix}[#{i}]", v, opts)
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
          gr += edge(job_uuid(job), node, {:label => prefix}, opts)        
        end
      else
        m = collection_uuid(sp)
        if m
          gr += edge(job_uuid(job), m, {:label => prefix}, opts)
          gr += ProvenanceHelper::generate_provenance_edges(pdata, visited, m, opts)
        elsif opts[:all_script_parameters]
          #id = "#{job[:uuid]}_#{prefix}"
          gr += "\"#{sp}\" [label=\"#{sp}\"];\n"
          gr += edge(job_uuid(job), sp, {:label => prefix}, opts)
        end
      end
    end
    gr
  end

  def self.generate_provenance_edges(pdata, visited, uuid, opts)
    gr = ""
    m = ProvenanceHelper::collection_uuid(uuid)
    uuid = m if m

    uuid = uuid.intern if uuid

    if (not uuid) or uuid.empty? or visited[uuid]

      #puts "already visited #{uuid}"
      return ""
    end

    if not pdata[uuid] then 
      return ProvenanceHelper::describe_node(pdata, uuid)
    else
      visited[uuid] = true
    end

    #puts "visiting #{uuid}"

    if m  
      # uuid is a collection
      gr += ProvenanceHelper::describe_node(pdata, uuid)

      pdata.each do |k, job|
        if job[:output] == uuid.to_s
          gr += self.edge(uuid, job_uuid(job), {:label => "output"}, opts)
          gr += ProvenanceHelper::generate_provenance_edges(pdata, visited, job[:uuid], opts)
        end
        if job[:log] == uuid.to_s
          gr += edge(uuid, job_uuid(job), {:label => "log"}, opts)
          gr += ProvenanceHelper::generate_provenance_edges(pdata, visited, job[:uuid], opts)
        end
      end
    else
      # uuid is something else
      rsc = ArvadosBase::resource_class_for_uuid uuid.to_s

      if rsc == Job
        job = pdata[uuid]
        if job
          gr += ProvenanceHelper::script_param_edges(pdata, visited, job, "", job[:script_parameters], opts)
        end
      else
        gr += ProvenanceHelper::describe_node(pdata, uuid)
      end
    end

    pdata.each do |k, link|
      if link[:head_uuid] == uuid.to_s and link[:link_class] == "provenance"
        gr += ProvenanceHelper::describe_node(pdata, link[:tail_uuid])
        gr += edge(link[:head_uuid], link[:tail_uuid], {:label => link[:name], :href => "/links/#{link[:uuid]}"}, opts) 
        gr += ProvenanceHelper::generate_provenance_edges(pdata, visited, link[:tail_uuid], opts)
      end
    end

    #puts "finished #{uuid}"

    gr
  end

  def self.create_provenance_graph(pdata, uuid, opts={})
    require 'open3'
    
    gr = """strict digraph {
node [fontsize=8,shape=box];
edge [fontsize=8];"""

    if opts[:direction] == :bottom_up
      gr += "edge [dir=back];"
    end

    #puts "pdata is #{pdata}"

    visited = {}
    if uuid.respond_to? :each
      uuid.each do |u|
        gr += ProvenanceHelper::generate_provenance_edges(pdata, visited, u, opts)
      end
    else
      gr += ProvenanceHelper::generate_provenance_edges(pdata, visited, uuid, opts)
    end

    gr += "}"
    svg = ""

    #puts gr

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
end
