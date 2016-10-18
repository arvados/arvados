class ProxyWorkUnit < WorkUnit
  require 'time'

  attr_accessor :lbl
  attr_accessor :proxied
  attr_accessor :my_children
  attr_accessor :unreadable_children

  def initialize proxied, label
    @lbl = label
    @proxied = proxied
  end

  def label
    @lbl
  end

  def proxied
    @proxied
  end

  def uuid
    get(:uuid)
  end

  def modified_by_user_uuid
    get(:modified_by_user_uuid)
  end

  def owner_uuid
    get(:owner_uuid)
  end

  def created_at
    t = get(:created_at)
    t = Time.parse(t) if (t.is_a? String)
    t
  end

  def started_at
    t = get(:started_at)
    t = Time.parse(t) if (t.is_a? String)
    t
  end

  def modified_at
    t = get(:modified_at)
    t = Time.parse(t) if (t.is_a? String)
    t
  end

  def finished_at
    t = get(:finished_at)
    t = Time.parse(t) if (t.is_a? String)
    t
  end

  def state_label
    state = get(:state)
    if ["Running", "RunningOnServer", "RunningOnClient"].include? state
      "Running"
    elsif state == 'New'
      "Not started"
    else
      state
    end
  end

  def state_bootstrap_class
    state = state_label
    case state
    when 'Complete'
      'success'
    when 'Failed', 'Cancelled'
      'danger'
    when 'Running', 'RunningOnServer', 'RunningOnClient'
      'info'
    else
      'default'
    end
  end

  def success?
    state = state_label
    if state == 'Complete'
      true
    elsif state == 'Failed' or state == 'Cancelled'
      false
    else
      nil
    end
  end

  def child_summary
    done = 0
    failed = 0
    todo = 0
    running = 0
    children.each do |c|
      case c.state_label
      when 'Complete'
        done = done+1
      when 'Failed', 'Cancelled'
        failed = failed+1
      when 'Running'
        running = running+1
      else
        todo = todo+1
      end
    end

    summary = {}
    summary[:done] = done
    summary[:failed] = failed
    summary[:todo] = todo
    summary[:running] = running
    summary
  end

  def child_summary_str
    summary = child_summary
    summary_txt = ''

    if state_label == 'Running'
      done = summary[:done] || 0
      running = summary[:running] || 0
      failed = summary[:failed] || 0
      todo = summary[:todo] || 0
      total = done + running + failed + todo

      if total > 0
        summary_txt += "#{summary[:done]} #{'child'.pluralize(summary[:done])} done,"
        summary_txt += "#{summary[:failed]} failed,"
        summary_txt += "#{summary[:running]} running,"
        summary_txt += "#{summary[:todo]} pending"
      end
    end
    summary_txt
  end

  def progress
    state = state_label
    if state == 'Complete'
      return 1.0
    elsif state == 'Failed' or state == 'Cancelled'
      return 0.0
    end

    summary = child_summary
    return 0.0 if summary.nil?

    done = summary[:done] || 0
    running = summary[:running] || 0
    failed = summary[:failed] || 0
    todo = summary[:todo] || 0
    total = done + running + failed + todo
    if total > 0
      (done+failed).to_f / total
    else
      0.0
    end
  end

  def children
    []
  end

  def outputs
    []
  end

  def title
    "process"
  end

  def has_unreadable_children
    @unreadable_children
  end

  def walltime
    if state_label != "Queued"
      if started_at
        ((if finished_at then finished_at else Time.now() end) - started_at)
      end
    end
  end

  def cputime
    if state_label != "Queued"
      if started_at
        (runtime_constraints.andand[:min_nodes] || 1) * ((finished_at || Time.now()) - started_at)
      end
    end
  end

  def queuedtime
    if state_label == "Queued"
      Time.now - Time.parse(created_at.to_s)
    end
  end

  def is_running?
    state_label == 'Running'
  end

  def is_paused?
    state_label == 'Paused'
  end

  def is_finished?
    state_label.in? ["Complete", "Failed", "Cancelled"]
  end

  def is_failed?
    state_label == 'Failed'
  end

  def show_runtime
    runningtime = ApplicationController.helpers.determine_wallclock_runtime(if children.any? then children else [self] end)

    walltime = 0
    if started_at
      walltime = if finished_at then (finished_at - started_at) else (Time.now - started_at) end
    end

    resp = '<p>'

    if started_at
      resp << "This #{title} started at "
      resp << ApplicationController.helpers.render_localized_date(started_at)
      resp << ". It "
      if state_label == 'Complete'
        resp << "completed in "
      elsif state_label == 'Failed'
         resp << "failed after "
      else
        resp << "has been active for "
      end

      if walltime > runningtime
        resp << ApplicationController.helpers.render_time(walltime, false)
      else
       resp << ApplicationController.helpers.render_time(runningtime, false)
      end

      if finished_at
        resp << " at "
        resp << ApplicationController.helpers.render_localized_date(finished_at)
      end
      resp << "."
    else
      if state_label
        resp << "This #{title} is "
        resp << if state_label == 'Running' then 'active' else state_label.downcase end
        resp << "."
      end
    end

    if is_failed?
      resp << " Check the Log tab for more detail about why it failed."
    end
    resp << "</p>"

    resp << "<p>"
    if state_label
      resp << "It "
      if state_label == 'Running'
        resp << "has run"
      else
        resp << "ran"
      end
      resp << " for "

      cpu_time = 0
      if children.any?
        cpu_time = children.map { |c|
          if c.started_at
             (c.runtime_constraints.andand[:min_nodes] || 1) * ((c.finished_at || Time.now()) - c.started_at)
          else
            0
          end
        }.reduce(:+) || 0
      else
        if started_at
          cpu_time = (runtime_constraints.andand[:min_nodes] || 1) * ((finished_at || Time.now()) - started_at)
        end
      end

      resp << ApplicationController.helpers.render_time(runningtime, false)
      if (walltime - runningtime) > 0
        resp << "("
        resp << ApplicationController.helpers.render_time(walltime - runningtime, false)
        resp << "queued)"
      end
      if cpu_time == 0
        resp << "."
      else
        resp << " and used "
        resp << ApplicationController.helpers.render_time(cpu_time, false)
        resp << " of node allocation time ("
        resp << (cpu_time/runningtime).round(1).to_s
        resp << "&Cross; scaling)."
      end
    end
    resp << "</p>"

    resp
  end

  def log_object_uuids
    [uuid]
  end

  def live_log_lines(limit)
    Log.where(object_uuid: log_object_uuids).
      order("created_at DESC").
      limit(limit).
      select { |log| log.properties[:text].is_a? String }.
      reverse.
      flat_map { |log| log.properties[:text].split("\n") }
  end

  protected

  def get key, obj=@proxied
    if obj.respond_to? key
      obj.send(key)
    elsif obj.is_a?(Hash)
      obj[key] || obj[key.to_s]
    end
  end
end
