class PipelineInstancesController < ApplicationController

  def show
    pipelines = [@object]

    if params[:compare]
      PipelineInstance.where(uuid: params[:compare]).each do |p| pipelines << p end
    end

    count = {}    
    provenance = {}
    pips = {}
    n = 1

    pipelines.each do |p|
      collections = []

      p.components.each do |k, v|
        j = v[:job]

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

      Collection.where(uuid: collections).each do |c|
        uuid = c.uuid.intern
        provenance[uuid] = c
        pips[uuid] = 0 unless pips[uuid] != nil
        pips[uuid] |= n
      end
      
      n = n << 1
    end

    #puts pips

    @prov_svg = ProvenanceHelper::create_provenance_graph provenance, "provenance_svg", {
      :all_script_parameters => true, 
      :combine_jobs => :script_and_version,
      :script_version_nodes => true,
      :pips => pips }
  end

end
