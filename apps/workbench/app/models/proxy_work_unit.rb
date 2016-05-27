class ProxyWorkUnit < WorkUnit
  require 'time'

  attr_accessor :lbl
  attr_accessor :proxied
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
    elsif state == 'Failed'
      false
    else
      nil
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

  def runtime_constraints
    get(:runtime_constraints)
  end

  def children
    []
  end

  def title
    "work unit"
  end

  def has_unreadable_children
    self.unreadable_children
  end

  def readable?
    resource_class = ArvadosBase::resource_class_for_uuid(uuid)
    resource_class.where(uuid: [uuid]).first rescue nil
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
