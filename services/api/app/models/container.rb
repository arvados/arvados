class Container < ArvadosModel
  include HasUuid
  include KindAndEtag
  include CommonApiTemplate

  serialize :environment, Hash
  serialize :mounts, Hash
  serialize :runtime_constraints, Hash
  serialize :command, Array

  before_validation :fill_field_defaults
  before_validation :set_timestamps
  validates :command, :container_image, :output_path, :cwd, :presence => true
  validate :validate_change

  has_many :container_requests, :foreign_key => :container_uuid, :class_name => 'ContainerRequest', :primary_key => :uuid

  api_accessible :user, extend: :common do |t|
    t.add :command
    t.add :container_image
    t.add :cwd
    t.add :environment
    t.add :finished_at
    t.add :log
    t.add :mounts
    t.add :output
    t.add :output_path
    t.add :priority
    t.add :progress
    t.add :runtime_constraints
    t.add :started_at
    t.add :state
  end

  # Supported states for a container
  States =
    [
     (Queued = 'Queued'),
     (Running = 'Running'),
     (Complete = 'Complete'),
     (Cancelled = 'Cancelled')
    ]

  def fill_field_defaults
    if self.new_record?
      self.state ||= Queued
      self.environment ||= {}
      self.runtime_constraints ||= {}
      self.mounts ||= {}
      self.cwd ||= "."
      self.priority ||= 1
    end
  end

  protected

  def permission_to_create
    current_user.andand.is_admin
  end

  def permission_to_update
    current_user.andand.is_admin
  end

  def check_permitted_updates permitted_fields
    attribute_names.each do |field|
      if not permitted_fields.include? field.to_sym and self.send((field.to_s + "_changed?").to_sym)
        errors.add field, "Illegal update of field #{field}"
      end
    end
  end

  def set_timestamps
    if self.state_changed? and self.state == Running
      self.started_at ||= db_current_time
    end

    if self.state_changed? and [Complete, Cancelled].include? self.state
      self.finished_at ||= db_current_time
    end
  end

  def validate_change
    permitted = [:modified_at, :modified_by_user_uuid, :modified_by_client_uuid]

    if self.new_record?
      permitted.push :owner_uuid, :command, :container_image, :cwd, :environment,
                     :mounts, :output_path, :priority, :runtime_constraints, :state
    end

    case self.state
    when Queued
      # permit priority change only.
      if self.state_changed? and not self.state_was.nil?
        errors.add :state, "Can only go to Queued from nil"
      else
        permitted.push :priority
      end
    when Running
      if self.state_changed?
        if self.state_was == Queued
          permitted.push :state, :started_at
        else
          errors.add :state, "Can only go to Runinng from Queued"
        end
      else
        permitted.push :priority, :progress
      end
    when Complete
      if self.state_changed?
        if self.state_was == Running
          permitted.push :state, :finished_at, :output, :log
        else
          errors.add :state, "Cannot go from #{self.state_was} from #{self.state}"
        end
      end
    when Cancelled
      if self.state_changed?
        if self.state_was == Running
          permitted.push :state, :finished_at, :output, :log
        elsif self.state_was == Queued
          permitted.push :state, :finished_at
        else
          errors.add :state, "Cannot go from #{self.state_was} from #{self.state}"
        end
      end
    else
      errors.add :state, "Invalid state #{self.state}"
    end

    check_permitted_updates permitted
  end

  def validate_fields
  end

end
