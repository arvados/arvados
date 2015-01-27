class JobsController < ApplicationController
  include JobsHelper

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

  def rerun_job_with_options_popup
    respond_to do |format|
      format.js
      format.html
    end
  end

  def rerun_job_with_options
    job_info = JSON.parse params['job_info']

    @object = Job.new

    @object.script = job_info['script']
    @object.repository = job_info['repository']
    @object.nondeterministic = job_info['nondeterministic']
    @object.script_parameters = job_info['script_parameters']
    @object.runtime_constraints = job_info['runtime_constraints']
    @object.supplied_script_version = job_info['supplied_script_version']

    if 'use_latest' == params['script']
      @object.script_version = job_info['supplied_script_version']
    else
      @object.script_version = job_info['script_version']
    end

    @object.save!
    show
  end
end
