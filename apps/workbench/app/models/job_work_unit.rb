class JobWorkUnit < ProxyWorkUnit
  def children
    # Job tasks
    uuid = get(:uuid)
    tasks = JobTask.filter([['job_uuid', '=', uuid]]).results
    items = []
    tasks.each do |t|
      items << t.work_unit("task #{items.size}")
    end

    # Jobs submitted by this job  --  TBD

    items
  end

  def progress
    state = get(:state)
    if state == 'Complete'
      return 1.0
    end

    tasks_summary = get(:tasks_summary)
    failed = tasks_summary[:failed] || 0 rescue 0
    done = tasks_summary[:done] || 0 rescue 0
    running = tasks_summary[:running] || 0 rescue 0
    todo = tasks_summary[:todo] || 0 rescue 0
    if done + running + failed + todo > 0
      total_tasks = done + running + failed + todo
      (done+failed).to_f / total_tasks
    else
      0.0
    end
  end

  def docker_image
    get(:docker_image_locator)
  end

  def nondeterministic
    get(:nondeterministic)
  end

  def priority
    get(:priority)
  end

  def log_collection
    get(:log)
  end

  def output
    get(:output)
  end

  def can_cancel?
    true
  end

  def uri
    uuid = get(:uuid)
    "/jobs/#{uuid}"
  end

  def child_summary
    get(:tasks_summary)
  end

  def title
    "job"
  end
end
