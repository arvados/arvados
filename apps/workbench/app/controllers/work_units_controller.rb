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
end
