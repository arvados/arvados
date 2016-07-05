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
    @filters = @next_page_filters || @filters || []

    filters = @filters + [%w(uuid is_a) + [%w(arvados#pipelineInstance)]]
    pipelines = PipelineInstance.order(["created_at desc"]).filter(filters)

    filters = @filters + [%w(uuid is_a) + [%w(arvados#containerRequest)]] + [['requesting_container_uuid', '=', nil]]
    crs = ContainerRequest.order(["created_at desc"]).filter(filters)
    
    procs = {}
    pipelines.results.each { |pi| procs[pi] = pi.created_at }
    crs.results.each { |c| procs[c] = c.created_at }

    @objects = Hash[procs.sort_by {|key, value| value}].keys.reverse.first(@limit)

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
