module PipelineInstancesHelper
  def pipeline_jobs
    if @object.components[:steps].is_a? Array
      pipeline_jobs_oldschool
    elsif @object.components.is_a? Hash
      pipeline_jobs_newschool
    end
  end

  protected

  def pipeline_jobs_newschool
    ret = []
    i = -1
    @object.components.each do |cname, c|
      i += 1
      pj = {index: i, name: cname}
      pj[:job] = c[:job].is_a?(Hash) ? c[:job] : {}
      pj[:percent_done] = 0
      pj[:percent_running] = 0
      if pj[:job][:success]
        if pj[:job][:output]
          pj[:progress] = 1.0
          pj[:percent_done] = 100
        else
          pj[:progress] = 0.0
        end
      else
        if pj[:job][:tasks_summary]
          begin
            ts = pj[:job][:tasks_summary]
            denom = ts[:done].to_f + ts[:running].to_f + ts[:todo].to_f
            pj[:progress] = (ts[:done].to_f + ts[:running].to_f/2) / denom
            pj[:percent_done] = 100.0 * ts[:done].to_f / denom
            pj[:percent_running] = 100.0 * ts[:running].to_f / denom
            pj[:progress_detail] = "#{ts[:done]} done #{ts[:running]} run #{ts[:todo]} todo"
          rescue
            pj[:progress] = 0.5
            pj[:percent_done] = 0.0
            pj[:percent_running] = 100.0
          end
        else
          pj[:progress] = 0.0
        end
      end
      if pj[:job]
        if pj[:job][:success]
          pj[:result] = 'complete'
          pj[:complete] = true
          pj[:progress] = 1.0
        elsif pj[:job][:finished_at]
          pj[:result] = 'failed'
          pj[:failed] = true
        elsif pj[:job][:started_at]
          pj[:result] = 'running'
        else
          pj[:result] = 'queued'
        end
      end
      pj[:job_id] = pj[:job][:uuid]
      pj[:job_link] = link_to_if_orvos_object pj[:job][:uuid]
      pj[:script_version] = pj[:job][:script_version]
      pj[:output] = pj[:job][:output]
      pj[:finished_at] = (Time.parse(pj[:job][:finished_at]) rescue nil)
      pj[:progress_bar] = raw("<div class=\"progress\" style=\"width:100px\"><div class=\"bar bar-success\" style=\"width:#{pj[:percent_done]}%\"></div><div class=\"bar\" style=\"width:#{pj[:percent_running]}%\"></div></div>")
      pj[:output_link] = link_to_if_orvos_object pj[:output]
      ret << pj
    end
    ret
  end

  def pipeline_jobs_oldschool
    ret = []
    @object.components[:steps].each_with_index do |step, i|
      pj = {index: i, name: step[:name]}
      if step[:complete] and step[:complete] != 0
        if step[:output_data_locator]
          pj[:progress] = 1.0
        else
          pj[:progress] = 0.0
        end
      else
        if step[:progress] and
            (re = step[:progress].match /^(\d+)\+(\d+)\/(\d+)$/)
          pj[:progress] = (((re[1].to_f + re[2].to_f/2) / re[3].to_f) rescue 0.5)
        else
          pj[:progress] = 0.0
        end
        if step[:failed]
          pj[:result] = 'failed'
          pj[:failed] = true
        end
      end
      if step[:warehousejob]
        if step[:complete]
          pj[:result] = 'complete'
          pj[:complete] = true
          pj[:progress] = 1.0
        elsif step[:warehousejob][:finishtime]
          pj[:result] = 'failed'
          pj[:failed] = true
        elsif step[:warehousejob][:starttime]
          pj[:result] = 'running'
        else
          pj[:result] = 'queued'
        end
      end
      pj[:progress_detail] = (step[:progress] rescue nil)
      pj[:job_id] = (step[:warehousejob][:id] rescue nil)
      pj[:job_link] = pj[:job_id]
      pj[:script] = step[:function]
      pj[:script_version] = (step[:warehousejob][:revision] rescue nil)
      pj[:output] = step[:output_data_locator]
      pj[:finished_at] = (Time.parse(step[:warehousejob][:finishtime]) rescue nil)
      pj[:progress_bar] = raw("<div class=\"progress\" style=\"width:100px\"><div class=\"bar\" style=\"width:#{pj[:progress]*100}%\"></div></div>")
      pj[:output_link] = link_to_if_orvos_object pj[:output]
      ret << pj
    end
    ret
  end
end
