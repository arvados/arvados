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
  validates :command, :container_image, :output_path, :cwd, :presence => true
  validate :validate_change

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

  def validate_change
    permitted = [:owner_uuid]

    case self.state
    when Uncommitted
      # Permit updating most fields
      permitted.push :command, :container_count_max,
                     :container_image, :cwd, :description, :environment,
                     :filters, :mounts, :name, :output_path, :priority,
                     :properties, :requesting_container_uuid, :runtime_constraints,
                     :state

      if self.container_uuid_changed? and (current_user.andand.is_admin or self.container_uuid.nil?)
        permitted.push :container_uuid
      end

    when Committed
      # Can only update priority, container count.
      permitted.push :priority, :container_count_max

      if self.state_changed?
        if self.state_was == Uncommitted
          permitted.push :state
        else
          errors.add :state, "Can only go from Uncommitted to Committed"
        end
      end

      if self.container_uuid_changed? and current_user.andand.is_admin
        permitted.push :container_uuid
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

end
