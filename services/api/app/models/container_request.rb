require 'whitelist_update'

class ContainerRequest < ArvadosModel
  include HasUuid
  include KindAndEtag
  include CommonApiTemplate
  include WhitelistUpdate

  serialize :properties, Hash
  serialize :environment, Hash
  serialize :mounts, Hash
  serialize :runtime_constraints, Hash
  serialize :command, Array

  before_validation :fill_field_defaults, :if => :new_record?
  before_validation :set_container
  validates :command, :container_image, :output_path, :cwd, :presence => true
  validate :validate_change
  after_save :update_priority

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

  protected

  def fill_field_defaults
    self.state ||= Uncommitted
    self.environment ||= {}
    self.runtime_constraints ||= {}
    self.mounts ||= {}
    self.cwd ||= "."
    self.priority ||= 1
  end

  def set_container
    if self.state_changed?
      if self.state == Committed and (self.state_was == Uncommitted or self.state_was.nil?)
        act_as_system_user do
          self.container_uuid = Container.resolve(self).andand.uuid
          Link.create!(head_uuid: self.container_uuid,
                       tail_uuid: self.owner_uuid,
                       link_class: "permission",
                       name: "can_read")
        end
      end
    end
  end

  def validate_change
    permitted = [:owner_uuid]

    case self.state
    when Uncommitted
      # Permit updating most fields
      permitted.push :command, :container_count_max,
                     :container_image, :cwd, :description, :environment,
                     :filters, :mounts, :name, :output_path, :priority,
                     :properties, :requesting_container_uuid, :runtime_constraints,
                     :state, :container_uuid

    when Committed
      if container_uuid.nil?
        errors.add :container_uuid, "Has not been resolved to a container."
      end

      # Can update priority, container count.
      permitted.push :priority, :container_count_max

      if self.state_changed?
        if self.state_was == Uncommitted or self.state_was.nil?
          # Allow create-and-commit in a single operation.
          permitted.push :command, :container_count_max,
                         :container_image, :cwd, :description, :environment,
                         :filters, :mounts, :name, :output_path, :priority,
                         :properties, :requesting_container_uuid, :runtime_constraints,
                         :state, :container_uuid
        else
          errors.add :state, "Can only go from Uncommitted to Committed"
        end
      end

    when Final
      if self.state_changed?
        if self.state_was == Committed
          permitted.push :state
        else
          errors.add :state, "Can only go from Committed to Final"
        end
      else
        errors.add "Cannot update record in Final state"
      end

    else
      errors.add :state, "Invalid state #{self.state}"
    end

    check_update_whitelist permitted
  end

  def update_priority
    if self.state == Committed and self.priority_changed?
      c = Container.find_by_uuid self.container_uuid
      c.update_priority!
    end
  end

end
