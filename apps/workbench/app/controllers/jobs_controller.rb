class JobsController < ApplicationController

  def generate_provenance(jobs)
    return if params['tab_pane'] != "Provenance"

    nodes = []
    collections = []
    jobs.each do |j|
      nodes << j
      collections << j[:output]
      collections.concat(ProvenanceHelper::find_collections(j[:script_parameters]))
      nodes << {:uuid => j[:script_version]}
    end

    Collection.where(uuid: collections).each do |c|
      nodes << c
    end

    @svg = ProvenanceHelper::create_provenance_graph nodes, "provenance_svg", {
      :request => request,
      :all_script_parameters => true,
      :script_version_nodes => true}
  end

  def index
    @svg = ""
    if params[:uuid]
      @objects = Job.where(uuid: params[:uuid])
      generate_provenance(@objects)
    else
      @limit = 20
    end
    super
  end

  def show
    generate_provenance([@object])
    super
  end

  def index_pane_list
    if params[:uuid]
      %w(Recent Provenance)
    else
      %w(Recent)
    end
  end

  def show_pane_list
    %w(Attributes Provenance Metadata JSON API)
  end
end
