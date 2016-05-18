class ProxyWorkUnit < WorkUnit
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
    self.proxied[:uuid]
  end

  def modified_by_user_uuid
    self.proxied[:modified_by_user_uuid]
  end

  def created_at
    self.proxied[:created_at]
  end

  def started_at
    self.proxied[:started_at]
  end

  def finished_at
    self.proxied[:finished_at]
  end

  def state_label
    if ["Running", "RunningOnServer", "RunningOnClient"].include? self.proxied[:state].to_s
      "running"
    else
      self.proxied[:state].to_s.downcase
    end
  end

  def state_bootstrap_class
    case self.proxied[:state]
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
    if self.proxied[:state] == 'Complete'
      true
    elsif self.proxied[:state] == 'Failed'
      false
    else
      nil
    end
  end

  def parameters
    self.proxied[:script_parameters]
  end

  def script
    self.proxied[:script]
  end

  def script_repository
    self.proxied[:repository]
  end

  def script_version
    self.proxied[:script_version]
  end

  def supplied_script_version
    self.proxied[:supplied_script_version]
  end

  def runtime_constraints
    self.proxied[:runtime_constraints]
  end
end
