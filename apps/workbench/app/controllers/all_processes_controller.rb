class AllProcessesController < ApplicationController
  def render_index
    respond_to do |f|
      f.json {
        if params[:partial]
          @next_page_href = next_page_href(partial: params[:partial], filters: @filters.to_json)
          render json: {
            content: render_to_string(partial: "work_unit/show_#{params[:partial]}",
                                      formats: [:html]),
            next_page_href: @next_page_href
          }
        else
          render json: @objects
        end
      }
      f.html {
        render
      }
      f.js {
        render
      }
    end
  end

  def find_objects_for_index
    @limit = 20

    @filters = @next_page_filters || @filters || []

    procs = {}

    # get next page of pipeline_instances
    filters = @filters + [%w(uuid is_a) + [%w(arvados#pipelineInstance)]]
    pipelines = PipelineInstance.limit(@limit).order(["created_at desc"]).filter(filters)
    pipelines.results.each { |pi| procs[pi] = pi.created_at }

    # get next page of jobs
    filters = @filters + [%w(uuid is_a) + [%w(arvados#job)]]
    jobs = Job.limit(@limit).order(["created_at desc"]).filter(filters)
    jobs.results.each { |pi| procs[pi] = pi.created_at }

    # get next page of container_requests
    filters = @filters + [%w(uuid is_a) + [%w(arvados#containerRequest)]]
    crs = ContainerRequest.limit(@limit).order(["created_at desc"]).filter(filters)
    crs.results.each { |c| procs[c] = c.created_at }

    @objects = Hash[procs.sort_by {|key, value| value}].keys.reverse

    @next_page_filters = @filters.reject do |attr,op,val|
      (attr == 'created_at') or (attr == 'uuid' and op == 'not in')
    end

    if @objects.any?
      last_created_at = @objects.last.created_at

      last_uuids = []
      @objects.each do |obj|
        last_uuids << obj.uuid if obj.created_at.eql?(last_created_at)
      end

      @next_page_filters += [['created_at', '<=', last_created_at]]
      @next_page_filters += [['uuid', 'not in', last_uuids]]
      @next_page_href = url_for(partial: :all_processes_rows,
                                limit: @limit,
                                filters: @next_page_filters.to_json)
    else
      @next_page_href = nil
    end
  end

  def next_page_href with_params={}
    @next_page_href
  end
end
