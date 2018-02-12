# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

class ContainerRequestsController < ApplicationController
  skip_around_filter :require_thread_api_token, if: proc { |ctrl|
    Rails.configuration.anonymous_user_token and
    'show' == ctrl.action_name
  }

  def generate_provenance(cr)
    return if params['tab_pane'] != "Provenance"

    nodes = {cr[:uuid] => cr}
    child_crs = []
    col_uuids = []
    col_pdhs = []
    col_uuids << cr[:output_uuid] if cr[:output_uuid]
    col_pdhs += ProvenanceHelper::cr_input_pdhs(cr)

    # Search for child CRs
    if cr[:container_uuid]
      child_crs = ContainerRequest.where(requesting_container_uuid: cr[:container_uuid])

      child_crs.each do |child|
        nodes[child[:uuid]] = child
        col_uuids << child[:output_uuid] if child[:output_uuid]
        col_pdhs += ProvenanceHelper::cr_input_pdhs(child)
      end
    end

    output_cols = {} # Indexed by UUID
    input_cols = {} # Indexed by PDH
    output_pdhs = []

    # Batch requests to get all related collections
    # First fetch output collections by UUID.
    Collection.filter([['uuid', 'in', col_uuids.uniq]]).each do |c|
      output_cols[c[:uuid]] = c
      output_pdhs << c[:portable_data_hash]
    end
    # Then, get only input collections by PDH. There could be more than one collection
    # per PDH: the number of collections is used on the collection node label.
    Collection.filter(
      [['portable_data_hash', 'in', col_pdhs - output_pdhs]]).each do |c|
      if input_cols[c[:portable_data_hash]]
        input_cols[c[:portable_data_hash]] << c
      else
        input_cols[c[:portable_data_hash]] = [c]
      end
    end

    @svg = ProvenanceHelper::create_provenance_graph(
      nodes, "provenance_svg",
      {
        :request => request,
        :direction => :top_down,
        :output_collections => output_cols,
        :input_collections => input_cols,
        :cr_children_of => {
          cr[:uuid] => child_crs.select{|child| child[:uuid]},
        },
      })
  end

  def show_pane_list
    panes = %w(Status Log Provenance Advanced)
    if @object.andand.state == 'Uncommitted'
      panes = %w(Inputs) + panes - %w(Log Provenance)
    end
    panes
  end

  def show
    generate_provenance(@object)
    super
  end

  def cancel
    if @object.container_uuid
      c = Container.select(['state']).where(uuid: @object.container_uuid).first
      if c && c.state != 'Running'
        # If the container hasn't started yet, setting priority=0
        # leaves our request in "Committed" state and doesn't cancel
        # the container (even if no other requests are giving it
        # priority). To avoid showing this container request as "on
        # hold" after hitting the Cancel button, set state=Final too.
        @object.state = 'Final'
      end
    end
    @object.update_attributes! priority: 0
    if params[:return_to]
      redirect_to params[:return_to]
    else
      redirect_to @object
    end
  end

  def update
    @updates ||= params[@object.class.to_s.underscore.singularize.to_sym]
    input_obj = @updates[:mounts].andand[:"/var/lib/cwl/cwl.input.json"].andand[:content]
    if input_obj
      workflow = @object.mounts[:"/var/lib/cwl/workflow.json"][:content]
      get_cwl_inputs(workflow).each do |input_schema|
        if not input_obj.include? cwl_shortname(input_schema[:id])
          next
        end
        required, primary_type, param_id = cwl_input_info(input_schema)
        if input_obj[param_id] == ""
          input_obj[param_id] = nil
        elsif primary_type == "boolean"
          input_obj[param_id] = input_obj[param_id] == "true"
        elsif ["int", "long"].include? primary_type
          input_obj[param_id] = input_obj[param_id].to_i
        elsif ["float", "double"].include? primary_type
          input_obj[param_id] = input_obj[param_id].to_f
        elsif ["File", "Directory"].include? primary_type
          re = CollectionsHelper.match_uuid_with_optional_filepath(input_obj[param_id])
          if re
            c = Collection.find(re[1])
            input_obj[param_id] = {"class" => primary_type,
                                   "location" => "keep:#{c.portable_data_hash}#{re[4]}",
                                   "arv:collection" => input_obj[param_id]}
          end
        end
      end
    end
    params[:merge] = true
    begin
      super
    rescue => e
      flash[:error] = e.to_s
      show
    end
  end

  def copy
    src = @object

    @object = ContainerRequest.new

    # By default the copied CR won't be reusing containers, unless use_existing=true
    # param is passed.
    command = src.command
    if params[:use_existing]
      @object.use_existing = true
      # Pass the correct argument to arvados-cwl-runner command.
      if src.command[0] == 'arvados-cwl-runner'
        command = src.command - ['--disable-reuse']
        command.insert(1, '--enable-reuse')
      end
    else
      @object.use_existing = false
      # Pass the correct argument to arvados-cwl-runner command.
      if src.command[0] == 'arvados-cwl-runner'
        command = src.command - ['--enable-reuse']
        command.insert(1, '--disable-reuse')
      end
    end

    @object.command = command
    @object.container_image = src.container_image
    @object.cwd = src.cwd
    @object.description = src.description
    @object.environment = src.environment
    @object.mounts = src.mounts
    @object.name = src.name
    @object.output_path = src.output_path
    @object.priority = 1
    @object.properties[:template_uuid] = src.properties[:template_uuid]
    @object.runtime_constraints = src.runtime_constraints
    @object.scheduling_parameters = src.scheduling_parameters
    @object.state = 'Uncommitted'

    # set owner_uuid to that of source, provided it is a project and writable by current user
    current_project = Group.find(src.owner_uuid) rescue nil
    if (current_project && current_project.writable_by.andand.include?(current_user.uuid))
      @object.owner_uuid = src.owner_uuid
    end

    super
  end

  def index
    @limit = 20
    super
  end

end
