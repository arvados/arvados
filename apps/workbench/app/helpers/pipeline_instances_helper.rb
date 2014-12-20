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
    pj[:job_link] = link_to_if_arvados_object pj[:job][:uuid] if pj[:job]
    pj
  end

  # Merge (started_at, finished_at) time range into the list of time ranges in
  # timestamps (timestamps must be sorted and non-overlapping).
  # return the updated timestamps list.
  def merge_range timestamps, started_at, finished_at
    # in the comments below, 'i' is the entry in the timestamps array and 'j'
    # is the started_at, finished_at range which is passed in.
    timestamps.each_index do |i|
      if started_at
        if started_at >= timestamps[i][0] and finished_at <= timestamps[i][1]
          # 'j' started and ended during 'i'
          return timestamps
        end

        if started_at < timestamps[i][0] and finished_at >= timestamps[i][0] and finished_at <= timestamps[i][1]
          # 'j' started before 'i' and finished during 'i'
          # re-merge range between when 'j' started and 'i' finished
          finished_at = timestamps[i][1]
          timestamps.delete_at i
          return merge_range timestamps, started_at, finished_at
        end

        if started_at >= timestamps[i][0] and started_at <= timestamps[i][1]
          # 'j' started during 'i' and finished sometime after
          # move end time of 'i' back
          # re-merge range between when 'i' started and 'j' finished
          started_at = timestamps[i][0]
          timestamps.delete_at i
          return merge_range timestamps, started_at, finished_at
        end

        if finished_at < timestamps[i][0]
          # 'j' finished before 'i' started, so insert before 'i'
          timestamps.insert i, [started_at, finished_at]
          return timestamps
        end
      end
    end

    timestamps << [started_at, finished_at]
  end

  # Accept a list of objects with [:started_at] and [:finshed_at] keys and
  # merge overlapping ranges to compute the time spent running after periods of
  # overlapping execution are factored out.
  def determine_wallclock_runtime jobs
    timestamps = []
    jobs.each do |j|
      insert_at = 0
      started_at = j[:started_at]
      finished_at = (if j[:finished_at] then j[:finished_at] else Time.now end)
      if started_at
        timestamps = merge_range timestamps, started_at, finished_at
      end
    end
    timestamps.map { |t| t[1] - t[0] }.reduce(:+) || 0
  end

  protected

  def pipeline_jobs_newschool object
    ret = []
    i = -1

    jobuuids = object.components.values.map { |c|
      c[:job][:uuid] if c.is_a?(Hash) and c[:job].is_a?(Hash)
    }.compact
    job = {}
    jobuuids.each do |jobuuid|
      job[jobuuid] = Job.find?(jobuuid)
    end.compact

    object.components.each do |cname, c|
      i += 1
      pj = {index: i, name: cname}
      if not c.is_a?(Hash)
        ret << pj
        next
      end
      if c[:job] and c[:job][:uuid] and job[c[:job][:uuid]]
        pj[:job] = job[c[:job][:uuid]]
      elsif c[:job].is_a?(Hash)
        pj[:job] = c[:job]
        if pj[:job][:started_at].is_a? String
          pj[:job][:started_at] = Time.parse(pj[:job][:started_at])
        end
        if pj[:job][:finished_at].is_a? String
          pj[:job][:finished_at] = Time.parse(pj[:job][:finished_at])
        end
        # If necessary, figure out the state based on the other fields.
        pj[:job][:state] ||= if pj[:job][:cancelled_at]
                               "Cancelled"
                             elsif pj[:job][:success] == false
                               "Failed"
                             elsif pj[:job][:success] == true
                               "Complete"
                             elsif pj[:job][:running] == true
                               "Running"
                             else
                               "Queued"
                             end
      else
        pj[:job] = {}
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

      case pj[:job][:state]
        when 'Complete'
        pj[:result] = 'complete'
        pj[:labeltype] = 'success'
        pj[:complete] = true
        pj[:progress] = 1.0
      when 'Failed'
        pj[:result] = 'failed'
        pj[:labeltype] = 'danger'
        pj[:failed] = true
      when 'Cancelled'
        pj[:result] = 'cancelled'
        pj[:labeltype] = 'danger'
        pj[:failed] = true
      when 'Running'
        pj[:result] = 'running'
        pj[:labeltype] = 'primary'
      when 'Queued'
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
      pj[:finished_at] = pj[:job][:finished_at]
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

  MINUTE = 60
  HOUR = 60 * MINUTE
  DAY = 24 * HOUR

  def render_runtime duration, use_words, round_to_min=true
    days = 0
    hours = 0
    minutes = 0
    seconds = 0

    if duration >= DAY
      days = (duration / DAY).floor
      duration -= days * DAY
    end

    if duration >= HOUR
      hours = (duration / HOUR).floor
      duration -= hours * HOUR
    end

    if duration >= MINUTE
      minutes = (duration / MINUTE).floor
      duration -= minutes * MINUTE
    end

    seconds = duration.floor

    if round_to_min and seconds >= 30
      minutes += 1
    end

    if use_words
      s = []
      if days > 0 then
        s << "#{days} day#{'s' if days != 1}"
      end
      if hours > 0 then
        s << "#{hours} hour#{'s' if hours != 1}"
      end
      if minutes > 0 then
        s << "#{minutes} minute#{'s' if minutes != 1}"
      end
      if not round_to_min or s.size == 0
        s << "#{seconds} second#{'s' if seconds != 1}"
      end
      s = s * " "
    else
      s = ""
      if days > 0
        s += "#{days}<span class='time-label-divider'>d</span> "
      end

      if (hours > 0)
        s += "#{hours}<span class='time-label-divider'>h</span>"
      end

      s += "#{minutes}<span class='time-label-divider'>m</span>"

      if not round_to_min
        s += "#{seconds}<span class='time-label-divider'>s</span>"
      end
    end

    raw(s)
  end

end
