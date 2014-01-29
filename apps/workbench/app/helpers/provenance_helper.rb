class ProvenanceHelper
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
end
