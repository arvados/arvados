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

  def stderr_log_records(job_uuids)
    Log.where(event_type: 'stderr',
              object_uuid: job_uuids).order('id DESC').results
  end

end
