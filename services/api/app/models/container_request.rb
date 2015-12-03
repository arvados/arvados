class ContainerRequest < ArvadosModel
  include HasUuid
  include KindAndEtag
  include CommonApiTemplate

  serialize :properties, Hash
  serialize :environment, Hash
  serialize :mounts, Hash
  serialize :runtime_constraints, Hash
  serialize :command, Array

  before_create :set_state_before_save
  validate :validate_change_permitted
  validate :validate_status
  validate :validate_state_change

  api_accessible :user, extend: :common do |t|
    t.add :command
    t.add :container_count_max
    t.add :container_image
    t.add :container_uuid
    t.add :cwd
    t.add :description
    t.add :environment
    t.add :expires_at
    t.add :filters
    t.add :mounts
    t.add :name
    t.add :output_path
    t.add :priority
    t.add :properties
    t.add :requesting_container_uuid
    t.add :runtime_constraints
    t.add :state
  end

  # Supported states for a container request
  States =
    [
     (Uncommitted = 'Uncommitted'),
     (Committed = 'Committed'),
     (Final = 'Final'),
    ]

  def set_state_before_save
    self.state ||= Uncommitted
  end

  def validate_change_permitted
    if self.changed?
      ok = case self.state
           when nil
             true
           when Uncommitted
             true
           when Committed
             # only allow state and priority to change.
             not (self.command_changed? or
                  self.container_count_max_changed? or
                  self.container_image_changed? or
                  self.container_uuid_changed? or
                  self.cwd_changed? or
                  self.description_changed? or
                  self.environment_changed? or
                  self.expires_at_changed? or
                  self.filters_changed? or
                  self.mounts_changed? or
                  self.name_changed? or
                  self.output_path_changed? or
                  self.properties_changed? or
                  self.requesting_container_uuid_changed? or
                  self.runtime_constraints_changed?)
           when Final
             false
           else
             false
           end
      if not ok
        errors.add :state, "Invalid update of container request in #{self.state} state"
      end
    end
  end

  def validate_status
    if self.state.in?(States)
      true
    else
      errors.add :state, "#{state.inspect} must be one of: #{States.inspect}"
      false
    end
  end

  def validate_state_change
    ok = true
    if self.state_changed?
      ok = case self.state_was
           when nil
             # Must go to Uncommitted
             self.state == Uncommitted
           when Uncommitted
             # Must go to Committed
             self.state == Committed
           when Committed
             # Must to go Final
             self.state == Final
           when Final
             # Once in a final state, don't permit any more state changes
             false
           else
             # Any other state transition is also invalid
             false
           end
      if not ok
        errors.add :state, "invalid change from #{self.state_was} to #{self.state}"
      end
    end
    ok
  end


end
