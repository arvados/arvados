class JobsController < ApplicationController
  def index
    @svg = ""
    if params[:uuid]
        @jobs = Job.where(uuid: params[:uuid])
        nodes = []
        collections = []
        @jobs.each do |j|
          nodes << j
          collections << j[:output]
          collections.concat(ProvenanceHelper::find_collections(j[:script_parameters]))
          nodes << {:uuid => j[:script_version]}
        end

        Collection.where(uuid: collections).each do |c|
          nodes << c
        end

      @svg = ProvenanceHelper::create_provenance_graph(nodes, {:all_script_parameters => true, :script_version_nodes => true})
    else
      @jobs = Job.all
    end
  end
end
