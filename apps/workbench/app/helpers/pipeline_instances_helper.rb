module PipelineInstancesHelper

  def pipeline_jobs object=nil
    object ||= @object
    if object.components[:steps].is_a? Array
      pipeline_jobs_oldschool object
    elsif object.components.is_a? Hash
      pipeline_jobs_newschool object
    end
  end

  def render_pipeline_jobs
    pipeline_jobs.collect do |pj|
      render_pipeline_job pj
    end
  end

  def render_pipeline_job pj
    pj[:progress_bar] = render partial: 'job_progress', locals: {:j => pj[:job]}
    pj[:output_link] = link_to_if_arvados_object pj[:output]
    pj[:job_link] = link_to_if_arvados_object pj[:job][:uuid]
    pj
  end

  protected

  def pipeline_jobs_newschool object
    ret = []
    i = -1

    jobuuids = object.components.values.map { |c|
      c[:job][:uuid] if c.is_a?(Hash) and c[:job].is_a?(Hash)
    }.compact
    job = {}
    Job.where(uuid: jobuuids).each do |j|
      job[j[:uuid]] = j
    end

    object.components.each do |cname, c|
      i += 1
      pj = {index: i, name: cname}
      if not c.is_a?(Hash)
        ret << pj
        next
      end
      if c[:job] and c[:job][:uuid] and job[c[:job][:uuid]]
        pj[:job] = job[c[:job][:uuid]]
      else
        pj[:job] = c[:job].is_a?(Hash) ? c[:job] : {}
      end
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
      if pj[:job][:success]
        pj[:result] = 'complete'
        pj[:labeltype] = 'success'
        pj[:complete] = true
        pj[:progress] = 1.0
      elsif pj[:job][:finished_at]
        pj[:result] = 'failed'
        pj[:labeltype] = 'danger'
        pj[:failed] = true
      elsif pj[:job][:started_at]
        pj[:result] = 'running'
        pj[:labeltype] = 'primary'
      elsif pj[:job][:uuid]
        pj[:result] = 'queued'
        pj[:labeltype] = 'default'
      else
        pj[:result] = 'none'
        pj[:labeltype] = 'default'
      end

      pj[:job_id] = pj[:job][:uuid]
      pj[:script] = pj[:job][:script] || c[:script]
      pj[:repository] = pj[:job][:script] || c[:repository]
      pj[:script_parameters] = pj[:job][:script_parameters] || c[:script_parameters]
      pj[:script_version] = pj[:job][:script_version] || c[:script_version]
      pj[:nondeterministic] = pj[:job][:nondeterministic] || c[:nondeterministic]
      pj[:output] = pj[:job][:output]
      pj[:output_uuid] = c[:output_uuid]
      pj[:finished_at] = (Time.parse(pj[:job][:finished_at]) rescue nil)
      ret << pj
    end
    ret
  end

  def pipeline_jobs_oldschool object
    ret = []
    object.components[:steps].each_with_index do |step, i|
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
      ret << pj
    end
    ret
  end

  def runtime duration, long
    hours = 0
    minutes = 0
    seconds = 0
    if duration >= 3600
      hours = (duration / 3600).floor
      duration -= hours * 3600
    end
    if duration >= 60
      minutes = (duration / 60).floor
      duration -= minutes * 60
    end
    duration = duration.floor

    if long
      s = ""
      if hours > 0 then
        s += "#{hours} hour#{'s' if hours != 1} "
      end
      if minutes > 0 then
        s += "#{minutes} minute#{'s' if minutes != 1} "
      end
      s += "#{duration} second#{'s' if duration != 1}"
    else
      s = "#{hours}:#{minutes.to_s.rjust(2, '0')}:#{duration.to_s.rjust(2, '0')}"
    end
    s
  end
end
