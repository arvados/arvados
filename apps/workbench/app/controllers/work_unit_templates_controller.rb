class WorkUnitTemplatesController < ApplicationController
  def find_objects_for_index
    return if !params[:partial]

    @limit = 40
    @filters = @filters || []

    # get next page of pipeline_templates
    filters = @filters + [["uuid", "is_a", ["arvados#pipelineTemplate"]]]
    pipelines = PipelineTemplate.limit(@limit).order(["created_at desc"]).filter(filters)

    # get next page of workflows
    filters = @filters + [["uuid", "is_a", ["arvados#workflow"]]]
    workflows = Workflow.limit(@limit).order(["created_at desc"]).filter(filters)

    @objects = (pipelines.to_a + workflows.to_a).sort_by(&:created_at).reverse.first(@limit)

    if @objects.any?
      @next_page_filters = next_page_filters('<=')
      @next_page_href = url_for(partial: :choose_rows,
                                filters: @next_page_filters.to_json)
    else
      @next_page_href = nil
    end
  end

  def next_page_href with_params={}
    @next_page_href
  end
end
