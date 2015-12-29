require 'whitelist_update'

class Container < ArvadosModel
  include HasUuid
  include KindAndEtag
  include CommonApiTemplate
  include WhitelistUpdate

  serialize :environment, Hash
  serialize :mounts, Hash
  serialize :runtime_constraints, Hash
  serialize :command, Array

  before_validation :fill_field_defaults, :if => :new_record?
  before_validation :set_timestamps
  validates :command, :container_image, :output_path, :cwd, :priority, :presence => true
  validate :validate_state_change
  validate :validate_change
  after_save :handle_completed

  has_many :container_requests, :foreign_key => :container_uuid, :class_name => 'ContainerRequest', :primary_key => :uuid

  api_accessible :user, extend: :common do |t|
    t.add :command
    t.add :container_image
    t.add :cwd
    t.add :environment
    t.add :exit_code
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

  State_transitions = {
    nil => [Queued],
    Queued => [Running, Cancelled],
    Running => [Complete, Cancelled]
  }

  def state_transitions
    State_transitions
  end

  def update_priority!
    if [Queued, Running].include? self.state
      # Update the priority of this container to the maximum priority of any of
      # its committed container requests and save the record.
      max = 0
      ContainerRequest.where(container_uuid: uuid).each do |cr|
        if cr.state == ContainerRequest::Committed and cr.priority > max
          max = cr.priority
        end
      end
      self.priority = max
      self.save!
    end
  end

  protected

  def fill_field_defaults
    self.state ||= Queued
    self.environment ||= {}
    self.runtime_constraints ||= {}
    self.mounts ||= {}
    self.cwd ||= "."
    self.priority ||= 1
  end

  def permission_to_create
    current_user.andand.is_admin
  end

  def permission_to_update
    current_user.andand.is_admin
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
    permitted = []

    if self.new_record?
      permitted.push :owner_uuid, :command, :container_image, :cwd, :environment,
                     :mounts, :output_path, :priority, :runtime_constraints, :state
    end

    case self.state
    when Queued
      # permit priority change only.
      permitted.push :priority

    when Running
      if self.state_changed?
        # At point of state change, can set state and started_at
        permitted.push :state, :started_at
      else
        # While running, can update priority and progress.
        permitted.push :priority, :progress
      end

    when Complete
      if self.state_changed?
        permitted.push :state, :finished_at, :output, :log, :exit_code
      else
        errors.add :state, "cannot update record"
      end

    when Cancelled
      if self.state_changed?
        if self.state_was == Running
          permitted.push :state, :finished_at, :output, :log
        elsif self.state_was == Queued
          permitted.push :state, :finished_at
        end
      else
        errors.add :state, "cannot update record"
      end

    else
      errors.add :state, "invalid state"
    end

    check_update_whitelist permitted
  end

  def handle_completed
    # This container is finished so finalize any associated container requests
    # that are associated with this container.
    if self.state_changed? and [Complete, Cancelled].include? self.state
      act_as_system_user do
        # Notify container requests associated with this container
        ContainerRequest.where(container_uuid: uuid,
                               :state => ContainerRequest::Committed).each do |cr|
          cr.container_completed!
        end

        # Try to cancel any outstanding container requests made by this container.
        ContainerRequest.where(requesting_container_uuid: uuid,
                               :state => ContainerRequest::Committed).each do |cr|
          cr.priority = 0
          cr.save
        end
      end
    end
  end

end
