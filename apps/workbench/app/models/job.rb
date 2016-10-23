class Job < ArvadosBase
  def self.goes_in_projects?
    true
  end

  def content_summary
    "#{script} job"
  end

  def editable_attributes
    %w(description)
  end

  def default_name
    if script
      x = "\"#{script}\" job"
    else
      x = super
    end
    if finished_at
      x += " finished #{finished_at.strftime('%b %-d')}"
    elsif started_at
      x += " started #{started_at.strftime('%b %-d')}"
    elsif created_at
      x += " submitted #{created_at.strftime('%b %-d')}"
    end
  end

  def cancel
    arvados_api_client.api "jobs/#{self.uuid}/", "cancel", {}
  end

  def self.queue_size
    arvados_api_client.api("jobs/", "queue_size", {"_method"=> "GET"})[:queue_size] rescue 0
  end

  def self.queue
    arvados_api_client.unpack_api_response arvados_api_client.api("jobs/", "queue", {"_method"=> "GET"})
  end

  def textile_attributes
    [ 'description' ]
  end

  def stderr_log_query(limit=nil)
    query = Log.where(object_uuid: self.uuid).order("created_at DESC")
    query = query.limit(limit) if limit
    query
  end

  def stderr_log_lines(limit=2000)
    stderr_log_query(limit).results.reverse.
      flat_map { |log| log.properties[:text].split("\n") rescue [] }
  end

  def work_unit(label=nil)
    JobWorkUnit.new(self, label, self.uuid)
  end
end
