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
  validate :validate_state_change
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

  State_transitions = {
    nil => [Uncommitted, Committed],
    Uncommitted => [Committed],
    Committed => [Final]
  }

  def state_transitions
    State_transitions
  end

  def skip_uuid_read_permission_check
    # XXX temporary until permissions are sorted out.
    %w(modified_by_client_uuid container_uuid requesting_container_uuid)
  end

  def container_completed!
    # may implement retry logic here in the future.
    self.state = ContainerRequest::Final
    self.save!
  end

  protected

  def fill_field_defaults
    self.state ||= Uncommitted
    self.environment ||= {}
    self.runtime_constraints ||= {}
    self.mounts ||= {}
    self.cwd ||= "."
  end

  # Turn a container request into a container.
  def resolve
    # In the future this will do things like resolve symbolic git and keep
    # references to content addresses.
    Container.create!({ :command => self.command,
                        :container_image => self.container_image,
                        :cwd => self.cwd,
                        :environment => self.environment,
                        :mounts => self.mounts,
                        :output_path => self.output_path,
                        :runtime_constraints => self.runtime_constraints })
  end

  def set_container
    if self.container_uuid_changed?
      if not current_user.andand.is_admin and not self.container_uuid.nil?
        errors.add :container_uuid, "can only be updated to nil."
      end
    else
      if self.state_changed?
        if self.state == Committed and (self.state_was == Uncommitted or self.state_was.nil?)
          act_as_system_user do
            self.container_uuid = self.resolve.andand.uuid
          end
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
        errors.add :container_uuid, "has not been resolved to a container."
      end

      if priority.nil?
        errors.add :priority, "cannot be nil"
      end

      # Can update priority, container count.
      permitted.push :priority, :container_count_max, :container_uuid

      if self.state_changed?
        # Allow create-and-commit in a single operation.
        permitted.push :command, :container_image, :cwd, :description, :environment,
                       :filters, :mounts, :name, :output_path, :properties,
                       :requesting_container_uuid, :runtime_constraints,
                       :state, :container_uuid
      end

    when Final
      if not current_user.andand.is_admin
        errors.add :state, "of container request can only be set to Final by system."
      end

      if self.state_changed?
          permitted.push :state
      else
        errors.add :state, "does not allow updates"
      end

    else
      errors.add :state, "invalid value"
    end

    check_update_whitelist permitted
  end

  def update_priority
    if [Committed, Final].include? self.state and (self.state_changed? or
                                                   self.priority_changed? or
                                                   self.container_uuid_changed?)
      [self.container_uuid_was, self.container_uuid].each do |cuuid|
        unless cuuid.nil?
          c = Container.find_by_uuid cuuid
          act_as_system_user do
            c.update_priority!
          end
        end
      end
    end
  end

end
