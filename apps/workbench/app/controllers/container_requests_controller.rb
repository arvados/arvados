class ContainerRequestsController < ApplicationController
  skip_around_filter :require_thread_api_token, if: proc { |ctrl|
    Rails.configuration.anonymous_user_token and
    'show' == ctrl.action_name
  }

  def show_pane_list
    panes = %w(Status Log Advanced)
    if @object.andand.state == 'Uncommitted'
      panes = %w(Inputs) + panes - %w(Log)
    end
    panes
  end

  def cancel
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

    # If "no reuse" requested, pass the correct argument to arvados-cwl-runner command.
    if params[:no_reuse] and src.command[0] == 'arvados-cwl-runner'
      command = src.command - ['--enable-reuse']
      command.insert(1, '--disable-reuse')
    else
      command = src.command
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
    @object.use_existing = false

    # set owner_uuid to that of source, provided it is a project and writable by current user
    current_project = Group.find(src.owner_uuid) rescue nil
    if (current_project && current_project.writable_by.andand.include?(current_user.uuid))
      @object.owner_uuid = src.owner_uuid
    end

    super
  end
end
