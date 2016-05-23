class ProxyWorkUnit < WorkUnit
  require 'time'

  attr_accessor :lbl
  attr_accessor :proxied

  def initialize proxied, label
    self.lbl = label
    self.proxied = proxied
  end

  def label
    self.lbl
  end

  def uuid
    (self.proxied.uuid if self.proxied.respond_to?(:uuid)) || self.proxied[:uuid]
  end

  def modified_by_user_uuid
    (self.proxied.modified_by_user_uuid if self.proxied.respond_to?(:modified_by_user_uuid)) || self.proxied[:modified_by_user_uuid]
  end

  def created_at
    t= (self.proxied.created_at if self.proxied.respond_to?(:created_at)) || self.proxied[:created_at]
    t.to_datetime if t
  end

  def started_at
    t = (self.proxied.started_at if self.proxied.respond_to?(:started_at)) || self.proxied[:started_at]
    t.to_datetime if t
  end

  def finished_at
    t = (self.proxied.finished_at if self.proxied.respond_to?(:finished_at)) || self.proxied[:finished_at]
    t.to_datetime if t
  end

  def state_label
    state = (self.proxied.state if self.proxied.respond_to?(:state)) || self.proxied[:state]
    if ["Running", "RunningOnServer", "RunningOnClient"].include? state
      "Running"
    else
      state
    end
  end

  def state_bootstrap_class
    state = (self.proxied.state if self.proxied.respond_to?(:state)) || self.proxied[:state]
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
    state = (self.proxied.state if self.proxied.respond_to?(:state)) || self.proxied[:state]
    if state == 'Complete'
      true
    elsif state == 'Failed'
      false
    else
      nil
    end
  end

  def parameters
    (self.proxied.script_parameters if self.proxied.respond_to?(:script_parameters)) || self.proxied[:script_parameters]
  end

  def repository
    (self.proxied.repository if self.proxied.respond_to?(:repository)) || self.proxied[:repository]
  end

  def script
    (self.proxied.script if self.proxied.respond_to?(:script)) || self.proxied[:script]
  end

  def script_version
    (self.proxied.send(:script_version) if self.proxied.respond_to?(:script_version)) || self.proxied[:script_version]
  end

  def supplied_script_version
    (self.proxied.supplied_script_version if self.proxied.respond_to?(:supplied_script_version)) || self.proxied[:supplied_script_version]
  end

  def runtime_constraints
    (self.proxied.runtime_constraints if self.proxied.respond_to?(:runtime_constraints)) || self.proxied[:runtime_constraints]
  end

  def children
    []
  end

  def title
    "work unit"
  end
end
