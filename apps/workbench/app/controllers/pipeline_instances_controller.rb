class PipelineInstancesController < ApplicationController
  skip_before_filter :find_object_by_uuid, only: :compare
  before_filter :find_objects_by_uuid, only: :compare
  include PipelineInstancesHelper
  include PipelineComponentsHelper

  def copy
    template = PipelineTemplate.find?(@object.pipeline_template_uuid)

    source = @object
    @object = PipelineInstance.new
    @object.pipeline_template_uuid = source.pipeline_template_uuid

    if params['components'] == 'use_latest' and template
      @object.components = template.components.deep_dup
      @object.components.each do |cname, component|
        # Go through the script parameters of each component
        # that are marked as user input and copy them over.
        # Skip any components that are not present in the
        # source instance (there's nothing to copy)
        if source.components.include? cname
          component[:script_parameters].each do |pname, val|
            if val.is_a? Hash and val[:dataclass]
              # this is user-inputtable, so check the value from the source pipeline
              srcvalue = source.components[cname][:script_parameters][pname]
              if not srcvalue.nil?
                component[:script_parameters][pname] = srcvalue
              end
            end
          end
        end
      end
    else
      @object.components = source.components.deep_dup
    end

    if params['script'] == 'use_same'
      # Go through each component and copy the script_version from each job.
      @object.components.each do |cname, component|
        if source.components.include? cname and source.components[cname][:job]
          component[:script_version] = source.components[cname][:job][:script_version]
        end
      end
    end

    @object.components.each do |cname, component|
      component.delete :job
    end
    @object.state = 'New'

    # set owner_uuid to that of source, provided it is a project and wriable by current user
    current_project = Group.find(source.owner_uuid) rescue nil
    if (current_project && current_project.writable_by.andand.include?(current_user.uuid))
      @object.owner_uuid = source.owner_uuid
    end

    super
  end

  def update
    @updates ||= params[@object.class.to_s.underscore.singularize.to_sym]
    if (components = @updates[:components])
      components.each do |cname, component|
        if component[:script_parameters]
          component[:script_parameters].each do |param, value_info|
            if value_info.is_a? Hash
              value_info_partitioned = value_info[:value].partition('/') if value_info[:value].andand.class.eql?(String)
              value_info_value = value_info_partitioned ? value_info_partitioned[0] : value_info[:value]
              value_info_class = resource_class_for_uuid value_info_value
              if value_info_class == Link
                # Use the link target, not the link itself, as script
                # parameter; but keep the link info around as well.
                link = Link.find value_info[:value]
                value_info[:value] = link.head_uuid
                value_info[:link_uuid] = link.uuid
                value_info[:link_name] = link.name
              else
                # Delete stale link_uuid and link_name data.
                value_info[:link_uuid] = nil
                value_info[:link_name] = nil
              end
              if value_info_class == Collection
                # to ensure reproducibility, the script_parameter for a
                # collection should be the portable_data_hash
                # keep the collection name and uuid for human-readability
                obj = Collection.find value_info_value
                if value_info_partitioned
                  value_info[:value] = obj.portable_data_hash + value_info_partitioned[1] + value_info_partitioned[2]
                  value_info[:selection_name] = obj.name + value_info_partitioned[1] + value_info_partitioned[2]
                else
                  value_info[:value] = obj.portable_data_hash
                  value_info[:selection_name] = obj.name
                end
                value_info[:selection_uuid] = obj.uuid
              end
            end
          end
        end
      end
    end
    super
  end

  def graph(pipelines)
    return nil, nil if params['tab_pane'] != "Graph"

    provenance = {}
    pips = {}
    n = 1

    # When comparing more than one pipeline, "pips" stores bit fields that
    # indicates which objects are part of which pipelines.

    pipelines.each do |p|
      collections = []
      hashes = []
      jobs = []

      p[:components].each do |k, v|
        provenance["component_#{p[:uuid]}_#{k}"] = v

        collections << v[:output_uuid] if v[:output_uuid]
        jobs << v[:job][:uuid] if v[:job]
      end

      jobs = jobs.compact.uniq
      if jobs.any?
        Job.where(uuid: jobs).each do |j|
          job_uuid = j.uuid

          provenance[job_uuid] = j
          pips[job_uuid] = 0 unless pips[job_uuid] != nil
          pips[job_uuid] |= n

          hashes << j[:output] if j[:output]
          ProvenanceHelper::find_collections(j) do |hash, uuid|
            collections << uuid if uuid
            hashes << hash if hash
          end

          if j[:script_version]
            script_uuid = j[:script_version]
            provenance[script_uuid] = {:uuid => script_uuid}
            pips[script_uuid] = 0 unless pips[script_uuid] != nil
            pips[script_uuid] |= n
          end
        end
      end

      hashes = hashes.compact.uniq
      if hashes.any?
        Collection.where(portable_data_hash: hashes).each do |c|
          hash_uuid = c.portable_data_hash
          provenance[hash_uuid] = c
          pips[hash_uuid] = 0 unless pips[hash_uuid] != nil
          pips[hash_uuid] |= n
        end
      end

      collections = collections.compact.uniq
      if collections.any?
        Collection.where(uuid: collections).each do |c|
          collection_uuid = c.uuid
          provenance[collection_uuid] = c
          pips[collection_uuid] = 0 unless pips[collection_uuid] != nil
          pips[collection_uuid] |= n
        end
      end

      n = n << 1
    end

    return provenance, pips
  end

  def show
    @pipelines = [@object]

    if params[:compare]
      PipelineInstance.where(uuid: params[:compare]).each do |p|
        @pipelines << p
      end
    end

    provenance, pips = graph(@pipelines)
    if provenance
      @prov_svg = ProvenanceHelper::create_provenance_graph provenance, "provenance_svg", {
        :request => request,
        :all_script_parameters => true,
        :combine_jobs => :script_and_version,
        :pips => pips,
        :only_components => true,
        :no_docker => true,
        :no_log => true}
    end

    super
  end

  def compare
    @breadcrumb_page_name = 'compare'

    @rows = []          # each is {name: S, components: [...]}

    if params['tab_pane'] == "Compare" or params['tab_pane'].nil?
      # Build a table: x=pipeline y=component
      @objects.each_with_index do |pi, pi_index|
        pipeline_jobs(pi).each do |component|
          # Find a cell with the same name as this component but no
          # entry for this pipeline
          target_row = nil
          @rows.each_with_index do |row, row_index|
            if row[:name] == component[:name] and !row[:components][pi_index]
              target_row = row
            end
          end
          if !target_row
            target_row = {name: component[:name], components: []}
            @rows << target_row
          end
          target_row[:components][pi_index] = component
        end
      end

      @rows.each do |row|
        # Build a "normal" pseudo-component for this row by picking the
        # most common value for each attribute. If all values are
        # equally common, there is no "normal".
        normal = {}              # attr => most common value
        highscore = {}           # attr => how common "normal" is
        score = {}               # attr => { value => how common }
        row[:components].each do |pj|
          next if pj.nil?
          pj.each do |k,v|
            vstr = for_comparison v
            score[k] ||= {}
            score[k][vstr] = (score[k][vstr] || 0) + 1
            highscore[k] ||= 0
            if score[k][vstr] == highscore[k]
              # tie for first place = no "normal"
              normal.delete k
            elsif score[k][vstr] == highscore[k] + 1
              # more pipelines have v than anything else
              highscore[k] = score[k][vstr]
              normal[k] = vstr
            end
          end
        end

        # Add a hash in component[:is_normal]: { attr => is_the_value_normal? }
        row[:components].each do |pj|
          next if pj.nil?
          pj[:is_normal] = {}
          pj.each do |k,v|
            pj[:is_normal][k] = (normal.has_key?(k) && normal[k] == for_comparison(v))
          end
        end
      end
    end

    if params['tab_pane'] == "Graph"
      provenance, pips = graph(@objects)

      @pipelines = @objects

      if provenance
        @prov_svg = ProvenanceHelper::create_provenance_graph provenance, "provenance_svg", {
          :request => request,
          :all_script_parameters => true,
          :combine_jobs => :script_and_version,
          :script_version_nodes => true,
          :pips => pips }
      end
    end

    @object = @objects.first

    show
  end

  def show_pane_list
    panes = %w(Components Log Graph Advanced)
    if @object and @object.state.in? ['New', 'Ready']
      panes = %w(Inputs) + panes - %w(Log)
    end
    if not @object.components.values.any? { |x| x[:job] rescue false }
      panes -= ['Graph']
    end
    panes
  end

  def compare_pane_list
    %w(Compare Graph)
  end

  def index
    if params[:search].andand.length.andand > 0
      @select ||= PipelineInstance.columns.map(&:name)
      base_search = PipelineInstance.select(@select)
      @objects = base_search.where(any: ['contains', params[:search]]).
                              uniq { |pi| pi.uuid }
    end

    @limit = 20
    super
  end

  protected
  def for_comparison v
    if v.is_a? Hash or v.is_a? Array
      v.to_json
    else
      v.to_s
    end
  end

  def find_objects_by_uuid
    @objects = model_class.where(uuid: params[:uuids])
  end

end
