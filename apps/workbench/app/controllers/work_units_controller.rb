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

    @next_page_filters = @filters.reject do |attr,op,val|
      (attr == 'created_at') or (attr == 'uuid' and op == 'not in')
    end

    if @objects.any?
      last_created_at = @objects.last.created_at

      last_uuids = []
      @objects.each do |obj|
        last_uuids << obj.uuid if obj.created_at.eql?(last_created_at)
      end

      @next_page_filters += [['created_at', '<=', last_created_at.strftime("%Y-%m-%dT%H:%M:%S.%N%z")]]
      @next_page_filters += [['uuid', 'not in', last_uuids]]
      @next_page_href = url_for(partial: :all_processes_rows,
                                partial_path: 'work_units/',
                                filters: @next_page_filters.to_json)
      preload_links_for_objects(@objects.to_a)
    else
      @next_page_href = nil
    end
  end

  def next_page_href with_params={}
    @next_page_href
  end
end
