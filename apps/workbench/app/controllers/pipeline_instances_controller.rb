class PipelineInstancesController < ApplicationController
  skip_before_filter :find_object_by_uuid, only: :compare
  before_filter :find_objects_by_uuid, only: :compare
  include PipelineInstancesHelper

  def graph(pipelines)
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
    if @object.components.empty? and @object.pipeline_template_uuid
      template = PipelineTemplate.find(@object.pipeline_template_uuid)
      pipeline = {}
      template.components.each do |component_name, component_props|
        pipeline[component_name] = {}
        component_props.each do |k, v|
          if k == :script_parameters
            pipeline[component_name][:script_parameters] = {}
            v.each do |param_name, param_value|
              if param_value.is_a? Hash
                if param_value[:value]
                  pipeline[component_name][:script_parameters][param_name] = param_value[:value]
                elsif param_value[:default]
                  pipeline[component_name][:script_parameters][param_name] = param_value[:default]
                elsif param_value[:optional] != nil or param_value[:required] != nil or param_value[:dataclass] != nil
                    pipeline[component_name][:script_parameters][param_name] = ""
                else
                  pipeline[component_name][:script_parameters][param_name] = param_value
                end
              else
                pipeline[component_name][:script_parameters][param_name] = param_value
              end
            end
          else
            pipeline[component_name][k] = v
          end
        end
      end
      @object.components= pipeline
      @object.save
    end

    @pipelines = [@object]

    if params[:compare]
      PipelineInstance.where(uuid: params[:compare]).each do |p|
        @pipelines << p
      end
    end

    provenance, pips = graph(@pipelines)

    @prov_svg = ProvenanceHelper::create_provenance_graph provenance, "provenance_svg", {
      :all_script_parameters => true, 
      :combine_jobs => :script_and_version,
      :script_version_nodes => true,
      :pips => pips }
    super
  end

  def compare
    @breadcrumb_page_name = 'compare'

    @rows = []          # each is {name: S, components: [...]}

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

    provenance, pips = graph(@objects)

    @pipelines = @objects

    @prov_svg = ProvenanceHelper::create_provenance_graph provenance, "provenance_svg", {
      :all_script_parameters => true, 
      :combine_jobs => :script_and_version,
      :script_version_nodes => true,
      :pips => pips }
  end

  def show_pane_list
    %w(Components Graph Attributes Metadata JSON API)
  end

  def compare_pane_list 
    %w(Compare Graph)
  end 

  def update
    updates = params[@object.class.to_s.underscore.singularize.to_sym]
    if updates["components"]
      require 'deep_merge/rails_compat'
      updates["components"] = updates["components"].deeper_merge(@object.components)
    end
    super
  end

  def index
    @objects ||= model_class.limit(20).all
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
