class ProxyWorkUnit < WorkUnit
  require 'time'

  attr_accessor :lbl
  attr_accessor :proxied
  attr_accessor :my_children
  attr_accessor :unreadable_children

  def initialize proxied, label
    self.lbl = label
    self.proxied = proxied
  end

  def label
    self.lbl
  end

  def uuid
    get(:uuid)
  end

  def modified_by_user_uuid
    get(:modified_by_user_uuid)
  end

  def created_at
    t = get(:created_at)
    t = Time.parse(t) if (t.andand.class == String)
    t
  end

  def started_at
    t = get(:started_at)
    t = Time.parse(t) if (t.andand.class == String)
    t
  end

  def finished_at
    t = get(:finished_at)
    t = Time.parse(t) if (t.andand.class == String)
    t
  end

  def state_label
    state = get(:state)
    if ["Running", "RunningOnServer", "RunningOnClient"].include? state
      "Running"
    else
      state
    end
  end

  def state_bootstrap_class
    state = get(:state)
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
    state = get(:state)
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
    state = get(:state)
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

  def parameters
    get(:script_parameters)
  end

  def repository
    get(:repository)
  end

  def script
    get(:script)
  end

  def script_version
    get(:script_version)
  end

  def supplied_script_version
    get(:supplied_script_version)
  end

  def docker_image
    get(:docker_image_locator)
  end

  def nondeterministic
    get(:nondeterministic)
  end

  def runtime_constraints
    get(:runtime_constraints)
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

  def outputs
    items = []
    children.each do |c|
      items << c.output if c.output
    end
    if !items.any?
      items << get(:output) if get(:output)
    end
    items
  end

  def children
    []
  end

  def title
    "process"
  end

  def has_unreadable_children
    self.unreadable_children
  end

  def readable?
    resource_class = ArvadosBase::resource_class_for_uuid(uuid)
    resource_class.where(uuid: [uuid]).first rescue nil
  end

  def link_to_log
    if state_label.in? ["Complete", "Failed", "Cancelled"]
      lc = log_collection
      if lc
        logCollection = Collection.find? lc
        if logCollection
          ApplicationController.helpers.link_to("Log", "#{uri}#Log")
        else
          "Log unavailable"
        end
      end
    elsif state_label == "Running"
      if readable?
        ApplicationController.helpers.link_to("Log", "#{uri}#Log")
      else
        "Log unavailable"
      end
    end
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
        (runtime_constraints.andand[:min_nodes] || 1) *
             ((finished_at || Time.now()) - started_at)
      end
    end
  end

  def queuedtime
    if state_label == "Queued"
      Time.now - Time.parse(created_at.to_s)
    end
  end

  def show_child_summary
    if state_label == "Running"
      if child_summary
        child_summary_str
      end
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

  def can_be_canceled?
    state_label.in? ["Queued", "Running"] and can_cancel?
  end

  def ran_for_str
    ran_for = nil
    if state_label
      ran_for = "It "
      if state_label == 'Running'
        ran_for << "has run"
      else
        ran_for << "ran"
      end
      ran_for << " for"
    end
    ran_for
  end

  def started_and_active_for_str
    active_for = nil

    if started_at
      active_for_1 = "This #{title} started at "
      active_for_2 = "It "
      if state_label == 'Complete'
        active_for_2 << "completed in "
      elsif state_label == 'Failed'
        active_for_2 << "failed after "
      else
        active_for_2 << "has been active for "
      end
      [active_for_1, active_for_2]
    end
  end

  protected

  def get key
    if self.proxied.respond_to? key
      self.proxied.send(key)
    elsif self.proxied.is_a?(Hash)
      self.proxied[key]
    end
  end
end
