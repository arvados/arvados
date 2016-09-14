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
  validate :validate_lock
  after_validation :assign_auth
  before_save :sort_serialized_attrs
  after_save :handle_completed

  has_many :container_requests, :foreign_key => :container_uuid, :class_name => 'ContainerRequest', :primary_key => :uuid
  belongs_to :auth, :class_name => 'ApiClientAuthorization', :foreign_key => :auth_uuid, :primary_key => :uuid

  api_accessible :user, extend: :common do |t|
    t.add :command
    t.add :container_image
    t.add :cwd
    t.add :environment
    t.add :exit_code
    t.add :finished_at
    t.add :locked_by_uuid
    t.add :log
    t.add :mounts
    t.add :output
    t.add :output_path
    t.add :priority
    t.add :progress
    t.add :runtime_constraints
    t.add :started_at
    t.add :state
    t.add :auth_uuid
  end

  # Supported states for a container
  States =
    [
     (Queued = 'Queued'),
     (Locked = 'Locked'),
     (Running = 'Running'),
     (Complete = 'Complete'),
     (Cancelled = 'Cancelled')
    ]

  State_transitions = {
    nil => [Queued],
    Queued => [Locked, Cancelled],
    Locked => [Queued, Running, Cancelled],
    Running => [Complete, Cancelled]
  }

  def state_transitions
    State_transitions
  end

  def update_priority!
    if [Queued, Locked, Running].include? self.state
      # Update the priority of this container to the maximum priority of any of
      # its committed container requests and save the record.
      self.priority = ContainerRequest.
        where(container_uuid: uuid,
              state: ContainerRequest::Committed).
        maximum('priority')
      self.save!
    end
  end

  def self.find_reusable(attrs)
    candidates = Container.
      where('command = ?', attrs[:command].to_yaml).
      where('cwd = ?', attrs[:cwd]).
      where('environment = ?', self.deep_sort_hash(attrs[:environment]).to_yaml).
      where('output_path = ?', attrs[:output_path]).
      where('container_image = ?', attrs[:container_image]).
      where('mounts = ?', self.deep_sort_hash(attrs[:mounts]).to_yaml).
      where('runtime_constraints = ?', self.deep_sort_hash(attrs[:runtime_constraints]).to_yaml).
      where('state in (?)', [Queued, Locked, Running, Complete]).
      reject {|c| c.state == Complete and
              (c.exit_code != 0 or c.output.nil? or c.log.nil?)}
    if candidates.empty?
      nil
    elsif candidates.count == 1
      candidates.first
    else
      # Multiple candidates found, search for the best one:
      # The most recent completed container
      winner = candidates.select {|c| c.state == Complete}.
        sort_by {|c| c.finished_at}.last
      return winner if not winner.nil?
      # The running container that's most likely to finish sooner.
      winner = candidates.select {|c| c.state == Running}.
        sort {|a, b| [b.progress, a.started_at] <=> [a.progress, b.started_at]}.first
      return winner if not winner.nil?
      # The locked container that's most likely to start sooner.
      winner = candidates.select {|c| c.state == Locked}.
        sort {|a, b| [b.priority, a.created_at] <=> [a.priority, b.created_at]}.first
      return winner if not winner.nil?
      # The queued container that's most likely to start sooner.
      winner = candidates.select {|c| c.state == Queued}.
        sort {|a, b| [b.priority, a.created_at] <=> [a.priority, b.created_at]}.first
      return winner if not winner.nil?
    end
  end

  def lock
    with_lock do
      if self.state == Locked
        raise AlreadyLockedError
      end
      self.state = Locked
      self.save!
    end
  end

  def unlock
    with_lock do
      if self.state == Queued
        raise InvalidStateTransitionError
      end
      self.state = Queued
      self.save!
    end
  end

  def self.readable_by(*users_list)
    if users_list.select { |u| u.is_admin }.any?
      return self
    end
    user_uuids = users_list.map { |u| u.uuid }
    uuid_list = user_uuids + users_list.flat_map { |u| u.groups_i_can(:read) }
    uuid_list.uniq!
    permitted = "(SELECT head_uuid FROM links WHERE link_class='permission' AND tail_uuid IN (:uuids))"
    joins(:container_requests).
      where("container_requests.uuid IN #{permitted} OR "+
            "container_requests.owner_uuid IN (:uuids)",
            uuids: uuid_list)
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
    permitted = [:state]

    if self.new_record?
      permitted.push(:owner_uuid, :command, :container_image, :cwd,
                     :environment, :mounts, :output_path, :priority,
                     :runtime_constraints)
    end

    case self.state
    when Queued, Locked
      permitted.push :priority

    when Running
      permitted.push :priority, :progress
      if self.state_changed?
        permitted.push :started_at
      end

    when Complete
      if self.state_was == Running
        permitted.push :finished_at, :output, :log, :exit_code
      end

    when Cancelled
      case self.state_was
      when Running
        permitted.push :finished_at, :output, :log
      when Queued, Locked
        permitted.push :finished_at
      end

    else
      # The state_transitions check will add an error message for this
      return false
    end

    check_update_whitelist permitted
  end

  def validate_lock
    # If the Container is already locked by someone other than the
    # current api_client_auth, disallow all changes -- except
    # priority, which needs to change to reflect max(priority) of
    # relevant ContainerRequests.
    if locked_by_uuid_was
      if locked_by_uuid_was != Thread.current[:api_client_authorization].uuid
        check_update_whitelist [:priority]
      end
    end

    if [Locked, Running].include? self.state
      # If the Container was already locked, locked_by_uuid must not
      # changes. Otherwise, the current auth gets the lock.
      need_lock = locked_by_uuid_was || Thread.current[:api_client_authorization].uuid
    else
      need_lock = nil
    end

    # The caller can provide a new value for locked_by_uuid, but only
    # if it's exactly what we expect. This allows a caller to perform
    # an update like {"state":"Unlocked","locked_by_uuid":null}.
    if self.locked_by_uuid_changed?
      if self.locked_by_uuid != need_lock
        return errors.add :locked_by_uuid, "can only change to #{need_lock}"
      end
    end
    self.locked_by_uuid = need_lock
  end

  def assign_auth
    if self.auth_uuid_changed?
      return errors.add :auth_uuid, 'is readonly'
    end
    if not [Locked, Running].include? self.state
      # don't need one
      self.auth.andand.update_attributes(expires_at: db_current_time)
      self.auth = nil
      return
    elsif self.auth
      # already have one
      return
    end
    cr = ContainerRequest.
      where('container_uuid=? and priority>0', self.uuid).
      order('priority desc').
      first
    if !cr
      return errors.add :auth_uuid, "cannot be assigned because priority <= 0"
    end
    self.auth = ApiClientAuthorization.
      create!(user_id: User.find_by_uuid(cr.modified_by_user_uuid).id,
              api_client_id: 0)
  end

  def sort_serialized_attrs
    if self.environment_changed?
      self.environment = self.class.deep_sort_hash(self.environment)
    end
    if self.mounts_changed?
      self.mounts = self.class.deep_sort_hash(self.mounts)
    end
    if self.runtime_constraints_changed?
      self.runtime_constraints = self.class.deep_sort_hash(self.runtime_constraints)
    end
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
