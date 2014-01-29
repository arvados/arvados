class PipelineInstancesController < ApplicationController

  def show
    provenance = {}
    somejob = nil
    collections = []
    @object.components.each do |k, v|
      j = v[:job]
      somejob = j[:uuid]
      provenance[somejob.intern] = j
      collections << j[:output].intern
      j[:dependencies].each do |k|
        collections << k.intern
      end
    end

    Collection.where(uuid: collections).each do |c|
      #puts c.uuid
      provenance[c.uuid.intern] = c
    end

    PipelineInstance.where(uuid: @object.uuid).each do |u|
      @prov_svg = ProvenanceHelper::create_provenance_graph provenance, collections, {:all_script_parameters => true}
    end
  end

end
