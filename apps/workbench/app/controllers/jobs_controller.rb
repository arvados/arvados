class JobsController < ApplicationController
  skip_around_filter :require_thread_api_token, if: proc { |ctrl|
    Rails.configuration.anonymous_user_token and
    'show' == ctrl.action_name
  }

  def generate_provenance(jobs)
    return if params['tab_pane'] != "Provenance"

    nodes = {}
    collections = []
    hashes = []
    jobs.each do |j|
      nodes[j[:uuid]] = j
      hashes << j[:output]
      ProvenanceHelper::find_collections(j[:script_parameters]) do |hash, uuid|
        collections << uuid if uuid
        hashes << hash if hash
      end
      nodes[j[:script_version]] = {:uuid => j[:script_version]}
    end

    Collection.where(uuid: collections).each do |c|
      nodes[c[:portable_data_hash]] = c
    end

    Collection.where(portable_data_hash: hashes).each do |c|
      nodes[c[:portable_data_hash]] = c
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
      render_index
    else
      @limit = 20
      super
    end
  end

  def cancel
    @object.cancel
    if params[:return_to]
      redirect_to params[:return_to]
    else
      redirect_to @object
    end
  end

  def show
    generate_provenance([@object])
    super
  end

  def logs
    @logs = Log.select(%w(event_type object_uuid event_at properties))
               .order('event_at DESC')
               .filter([["event_type",  "=", "stderr"],
                        ["object_uuid", "in", [@object.uuid]]])
               .limit(500)
               .results
               .to_a
               .map{ |e| e.serializable_hash.merge({ 'prepend' => true }) }
    respond_to do |format|
      format.json { render json: @logs }
    end
  end

  def index_pane_list
    if params[:uuid]
      %w(Recent Provenance)
    else
      %w(Recent)
    end
  end

  def show_pane_list
    %w(Status Log Details Provenance Advanced)
  end
end
