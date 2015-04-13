module JobsHelper
  def stderr_log_history(job_uuids, limit=2000)
    results = []

    log_history = Log.where(event_type: 'stderr',
                            object_uuid: job_uuids).limit(limit).order('id DESC')
    if !log_history.results.empty?
      reversed_results = log_history.results.reverse
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

end
