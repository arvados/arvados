module JobsHelper
  def stderr_log_history(job_uuids)
    results = []

    log_history = stderr_log_records(job_uuids)
    if !log_history.empty?
      reversed_results = log_history.reverse
      reversed_results.each do |entry|
        if entry.andand.properties
          properties = entry.properties
          text = properties[:text]
          if text
            results = results.concat text.split("\n")
          end
        end
      end
    end
    return results
  end

  def stderr_log_records(job_uuids, extra_filters = nil)
    filters = [["event_type",  "=", "stderr"],
               ["object_uuid", "in", job_uuids]]
    filters += extra_filters if extra_filters
    last_entry = Log.select(%w(event_at)).order('id DESC').limit(1).filter(filters).results.first
    if last_entry
      filters += [["event_at", ">=", last_entry.event_at - 10.minutes]]
      Log.select(%w(event_type object_uuid event_at properties))
         .order('id DESC')
         .filter(filters)
         .results
    else
      []
    end
  end

end
