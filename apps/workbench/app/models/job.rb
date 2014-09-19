class Job < ArvadosBase
  def self.goes_in_projects?
    true
  end

  def content_summary
    "#{script} job"
  end

  def attribute_editable? attr, *args
    if attr.to_sym == :description
      super && attr.to_sym == :description
    else
      false
    end
  end

  def self.creatable?
    false
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

  def self.state job
    if job.respond_to? :state and job.state
      return job.state
    end

    if not job[:cancelled_at].nil?
      "Cancelled"
    elsif not job[:finished_at].nil? or not job[:success].nil?
      if job[:success]
        "Completed"
      else
        "Failed"
      end
    elsif job[:running]
      "Running"
    else
      "Queued"
    end
  end

  def textile_attributes
    [ 'description' ]
  end
end
