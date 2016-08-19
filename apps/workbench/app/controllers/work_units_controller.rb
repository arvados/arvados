class WorkUnitsController < ApplicationController
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
      if workflow.workflow
        begin
          wf_json = YAML::load(workflow.workflow)
        rescue => e
          logger.error "Error converting workflow yaml to json: #{e.message}"
          raise ArgumentError, "Error converting workflow yaml to json: #{e.message}"
        end
      end

      model_class = ContainerRequest

      attrs['name'] = "#{workflow['name']} container" if workflow['name'].present?
      attrs['properties'] = {'template_uuid' => template_uuid}
      attrs['priority'] = 1
      attrs['state'] = "Uncommitted"

      # required
      attrs['command'] = ["arvados-cwl-runner", "--local", "--api=containers", "/var/lib/cwl/workflow.json", "/var/lib/cwl/cwl.input.json"]
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
end
