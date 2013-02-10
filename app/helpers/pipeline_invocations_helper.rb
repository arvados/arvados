module PipelineInvocationsHelper
  def pipeline_jobs
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
      pj[:command] = step[:function]
      pj[:command_version] = (step[:warehousejob][:revision] rescue nil)
      pj[:output] = step[:output_data_locator]
      pj[:finished_at] = (Time.parse(step[:warehousejob][:finishtime]) rescue nil)
      pj[:progress_bar] = raw("<div class=\"progress\" style=\"width:100px\"><div class=\"bar\" style=\"width:#{pj[:progress]*100}%\"></div></div>")
      pj[:output_link] = link_to_if_orvos_object pj[:output]
      ret << pj
    end
    ret
  end
end
