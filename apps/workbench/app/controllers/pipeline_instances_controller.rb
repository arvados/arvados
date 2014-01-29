class PipelineInstancesController < ApplicationController

  def show
    provenance = {}
    somejob = nil
    collections = []
    @object.components.each do |k, v|
      j = v[:job]
      somejob = j[:uuid]
      provenance[somejob.intern] = j
      collections << j[:output]
    end

    puts collections
    puts '---'

    Collection.where(uuid: collections).each do |c|
      puts c.uuid
      provenance[c.uuid.intern] = c
    end

    PipelineInstance.where(uuid: @object.uuid).each do |u|
      @prov_svg = CollectionsController::create_provenance_graph provenance, somejob
    end
  end

end
