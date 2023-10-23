# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

class ContainerRequestsController < ApplicationController
  skip_around_action :require_thread_api_token, if: proc { |ctrl|
    !Rails.configuration.Users.AnonymousUserToken.empty? and
    'show' == ctrl.action_name
  }

  def generate_provenance(cr)
    return if params['tab_pane'] != "Provenance"

    nodes = {}
    child_crs = []
    col_uuids = []
    col_pdhs = []
    col_uuids << cr[:output_uuid] if cr[:output_uuid]
    col_pdhs += ProvenanceHelper::cr_input_pdhs(cr)

    # Search for child CRs
    if cr[:container_uuid]
      child_crs = ContainerRequest.where(requesting_container_uuid: cr[:container_uuid]).with_count("none")

      child_crs.each do |child|
        nodes[child[:uuid]] = child
        col_uuids << child[:output_uuid] if child[:output_uuid]
        col_pdhs += ProvenanceHelper::cr_input_pdhs(child)
      end
    end

    if nodes.length == 0
      nodes[cr[:uuid]] = cr
    end

    pdh_to_col = {} # Indexed by PDH
    output_pdhs = []

    # Batch requests to get all related collections
    # First fetch output collections by UUID.
    Collection.filter([['uuid', 'in', col_uuids.uniq]]).with_count("none").each do |c|
      output_pdhs << c[:portable_data_hash]
      pdh_to_col[c[:portable_data_hash]] = c
      nodes[c[:uuid]] = c
    end
    # Next, get input collections by PDH.
    Collection.filter(
      [['portable_data_hash', 'in', col_pdhs - output_pdhs]]).with_count("none").each do |c|
      nodes[c[:portable_data_hash]] = c
    end

    @svg = ProvenanceHelper::create_provenance_graph(
      nodes, "provenance_svg",
      {
        :request => request,
        :pdh_to_uuid => pdh_to_col,
      }
    )
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
      c = Container.select(['state']).where(uuid: @object.container_uuid).with_count("none").first
      if c && c.state != 'Running'
        # If the container hasn't started yet, setting priority=0
        # leaves our request in "Committed" state and doesn't cancel
        # the container (even if no other requests are giving it
        # priority). To avoid showing this container request as "on
        # hold" after hitting the Cancel button, set state=Final too.
        @object.state = 'Final'
      end
    end
    @object.update! priority: 0
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
                                   "http://arvados.org/cwl#collectionUUID" => re[1]}
          end
        end
      end
    end
    params[:merge] = true

    if !@updates[:reuse_steps].nil?
      if @updates[:reuse_steps] == "false"
        @updates[:reuse_steps] = false
      end
      @updates[:command] ||= @object.command
      @updates[:command] -= ["--disable-reuse", "--enable-reuse"]
      if @updates[:reuse_steps]
        @updates[:command].insert(1, "--enable-reuse")
      else
        @updates[:command].insert(1, "--disable-reuse")
      end
      @updates.delete(:reuse_steps)
    end

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

    # set owner_uuid to that of source, provided it is a project and writable by current user
    if params[:work_unit].andand[:owner_uuid]
      @object.owner_uuid = src.owner_uuid = params[:work_unit][:owner_uuid]
    else
      current_project = Group.find(src.owner_uuid) rescue nil
      if (current_project && current_project.writable_by.andand.include?(current_user.uuid))
        @object.owner_uuid = src.owner_uuid
      end
    end

    command = src.command
    if command[0] == 'arvados-cwl-runner'
      command.each_with_index do |arg, i|
        if arg.start_with? "--project-uuid="
          command[i] = "--project-uuid=#{@object.owner_uuid}"
        end
      end
      command -= ["--disable-reuse", "--enable-reuse"]
      command.insert(1, '--enable-reuse')
    end

    if params[:use_existing] == "false"
      params[:use_existing] = false
    elsif params[:use_existing] == "true"
      params[:use_existing] = true
    end

    if params[:use_existing] || params[:use_existing].nil?
      # If nil, reuse workflow steps but not the workflow runner.
      @object.use_existing = !!params[:use_existing]

      # Pass the correct argument to arvados-cwl-runner command.
      if command[0] == 'arvados-cwl-runner'
        command -= ["--disable-reuse", "--enable-reuse"]
        command.insert(1, '--enable-reuse')
      end
    else
      @object.use_existing = false
      # Pass the correct argument to arvados-cwl-runner command.
      if command[0] == 'arvados-cwl-runner'
        command -= ["--disable-reuse", "--enable-reuse"]
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

    super
  end

  def index
    @limit = 20
    super
  end

end
