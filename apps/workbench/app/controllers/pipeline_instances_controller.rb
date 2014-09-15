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
    else
      @object.components = source.components.deep_dup
    end

    if params['script'] == 'use_same'
      # Go through each component and copy the script_version from each job.
      @object.components.each do |cname, component|
        if source.components[cname][:job]
          component[:script_version] = source.components[cname][:job][:script_version]
        end
      end
    end

    @object.components.each do |cname, component|
      component.delete :job
    end
    @object.state = 'New'
    super
  end

  def update
    @updates ||= params[@object.class.to_s.underscore.singularize.to_sym]
    if (components = @updates[:components])
      components.each do |cname, component|
        if component[:script_parameters]
          component[:script_parameters].each do |param, value_info|
            if value_info.is_a? Hash
              if resource_class_for_uuid(value_info[:value]) == Link
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
            end
          end
        end
      end
    end
    super
  end

  def graph(pipelines)
    return nil, nil if params['tab_pane'] != "Graph"

    count = {}
    provenance = {}
    pips = {}
    n = 1

    pipelines.each do |p|
      collections = []

      p.components.each do |k, v|
        j = v[:job] || next

        uuid = j[:uuid].intern
        provenance[uuid] = j
        pips[uuid] = 0 unless pips[uuid] != nil
        pips[uuid] |= n

        collections << j[:output]
        ProvenanceHelper::find_collections(j[:script_parameters]).each do |k|
          collections << k
        end

        uuid = j[:script_version].intern
        provenance[uuid] = {:uuid => uuid}
        pips[uuid] = 0 unless pips[uuid] != nil
        pips[uuid] |= n
      end

      Collection.where(uuid: collections.compact).each do |c|
        uuid = c.uuid.intern
        provenance[uuid] = c
        pips[uuid] = 0 unless pips[uuid] != nil
        pips[uuid] |= n
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
        :script_version_nodes => true,
        :pips => pips }
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
