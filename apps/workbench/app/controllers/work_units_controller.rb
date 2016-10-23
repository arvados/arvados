class WorkUnitsController < ApplicationController
  skip_around_filter :require_thread_api_token, if: proc { |ctrl|
    Rails.configuration.anonymous_user_token and
    'show_child_component' == ctrl.action_name
  }

  def find_objects_for_index
    # If it's not the index rows partial display, just return
    # The /index request will again be invoked to display the
    # partial at which time, we will be using the objects found.
    return if !params[:partial]

    @limit = 20
    @filters = @filters || []

    # get next page of pipeline_instances
    filters = @filters + [["uuid", "is_a", ["arvados#pipelineInstance"]]]
    pipelines = PipelineInstance.limit(@limit).order(["created_at desc"]).filter(filters)

    # get next page of jobs
    filters = @filters + [["uuid", "is_a", ["arvados#job"]]]
    jobs = Job.limit(@limit).order(["created_at desc"]).filter(filters)

    # get next page of container_requests
    filters = @filters + [["uuid", "is_a", ["arvados#containerRequest"]]]
    crs = ContainerRequest.limit(@limit).order(["created_at desc"]).filter(filters)
    @objects = (jobs.to_a + pipelines.to_a + crs.to_a).sort_by(&:created_at).reverse.first(@limit)

    if @objects.any?
      @next_page_filters = next_page_filters('<=')
      @next_page_href = url_for(partial: :all_processes_rows,
                                filters: @next_page_filters.to_json)
      preload_links_for_objects(@objects.to_a)
    else
      @next_page_href = nil
    end
  end

  def next_page_href with_params={}
    @next_page_href
  end

  def create
    template_uuid = params['work_unit']['template_uuid']

    attrs = {}
    rc = resource_class_for_uuid(template_uuid)
    if rc == PipelineTemplate
      model_class = PipelineInstance
      attrs['pipeline_template_uuid'] = template_uuid
    elsif rc == Workflow
      # workflow json
      workflow = Workflow.find? template_uuid
      if workflow.definition
        begin
          wf_json = YAML::load(workflow.definition)
        rescue => e
          logger.error "Error converting definition yaml to json: #{e.message}"
          raise ArgumentError, "Error converting definition yaml to json: #{e.message}"
        end
      end

      model_class = ContainerRequest

      attrs['name'] = "#{workflow['name']} container" if workflow['name'].present?
      attrs['properties'] = {'template_uuid' => template_uuid}
      attrs['priority'] = 1
      attrs['state'] = "Uncommitted"

      # required
      attrs['command'] = ["arvados-cwl-runner", "--local", "--api=containers", "/var/lib/cwl/workflow.json#main", "/var/lib/cwl/cwl.input.json"]
      attrs['container_image'] = "arvados/jobs"
      attrs['cwd'] = "/var/spool/cwl"
      attrs['output_path'] = "/var/spool/cwl"

      # mounts
      mounts = {
        "/var/lib/cwl/cwl.input.json" => {
          "kind" => "json",
          "content" => {}
        },
        "stdout" => {
          "kind" => "file",
          "path" => "/var/spool/cwl/cwl.output.json"
        },
        "/var/spool/cwl" => {
          "kind" => "collection",
          "writable" => true
        }
      }
      if wf_json
        mounts["/var/lib/cwl/workflow.json"] = {
          "kind" => "json",
          "content" => wf_json
        }
      end
      attrs['mounts'] = mounts

      # runtime constriants
      runtime_constraints = {
        "vcpus" => 1,
        "ram" => 256000000,
        "API" => true
      }
      attrs['runtime_constraints'] = runtime_constraints
    else
      raise ArgumentError, "Unsupported template uuid: #{template_uuid}"
    end

    attrs['owner_uuid'] = params['work_unit']['owner_uuid']
    @object ||= model_class.new attrs

    if @object.save
      redirect_to @object
    else
      render_error status: 422
    end
  end

  def find_object_by_uuid
    if params['object_type']
      @object = params['object_type'].constantize.find(params['uuid'])
    else
      super
    end
  end

  def show_child_component
    data = JSON.load(params[:action_data])

    current_obj = {}
    current_obj_uuid = data['current_obj_uuid']
    current_obj_name = data['current_obj_name']
    current_obj_type = data['current_obj_type']
    current_obj_parent = data['current_obj_parent']
    if current_obj_uuid
      resource_class = resource_class_for_uuid current_obj_uuid
      obj = object_for_dataclass(resource_class, current_obj_uuid)
      current_obj = obj if obj
    end

    if current_obj.is_a?(Hash) and !current_obj.any?
      if current_obj_parent
        resource_class = resource_class_for_uuid current_obj_parent
        parent = object_for_dataclass(resource_class, current_obj_parent)
        parent_wu = parent.work_unit
        children = parent_wu.children
        if current_obj_uuid
          wu = children.select {|c| c.uuid == current_obj_uuid}.first
        else current_obj_name
          wu = children.select {|c| c.label.to_s == current_obj_name}.first
        end
      end
    else
      if current_obj_type == JobWorkUnit.to_s
        wu = JobWorkUnit.new(current_obj, current_obj_name, current_obj_parent)
      elsif current_obj_type == PipelineInstanceWorkUnit.to_s
        wu = PipelineInstanceWorkUnit.new(current_obj, current_obj_name, current_obj_parent)
      elsif current_obj_type == ContainerWorkUnit.to_s
        wu = ContainerWorkUnit.new(current_obj, current_obj_name, current_obj_parent)
      end
    end

    respond_to do |f|
      f.html { render(partial: "show_component", locals: {wu: wu}) }
    end
  end
end
