class Job < ArvadosModel
  include HasUuid
  include KindAndEtag
  include CommonApiTemplate
  attr_protected :docker_image_locator
  serialize :script_parameters, Hash
  serialize :runtime_constraints, Hash
  serialize :tasks_summary, Hash
  before_create :ensure_unique_submit_id
  after_commit :trigger_crunch_dispatch_if_cancelled, :on => :update
  before_validation :set_priority
  before_validation :update_state_from_old_state_attrs
  validate :ensure_script_version_is_commit
  validate :find_docker_image_locator
  validate :validate_status
  validate :validate_state_change
  after_validation :update_timestamps_when_state_changes

  has_many :commit_ancestors, :foreign_key => :descendant, :primary_key => :script_version
  has_many(:nodes, foreign_key: :job_uuid, primary_key: :uuid)

  class SubmitIdReused < StandardError
  end

  api_accessible :user, extend: :common do |t|
    t.add :submit_id
    t.add :priority
    t.add :script
    t.add :script_parameters
    t.add :script_version
    t.add :cancelled_at
    t.add :cancelled_by_client_uuid
    t.add :cancelled_by_user_uuid
    t.add :started_at
    t.add :finished_at
    t.add :output
    t.add :success
    t.add :running
    t.add :state
    t.add :is_locked_by_uuid
    t.add :log
    t.add :runtime_constraints
    t.add :tasks_summary
    t.add :dependencies
    t.add :nondeterministic
    t.add :repository
    t.add :supplied_script_version
    t.add :docker_image_locator
    t.add :queue_position
    t.add :node_uuids
    t.add :description
  end

  # Supported states for a job
  States = [
            (Queued = 'Queued'),
            (Running = 'Running'),
            (Cancelled = 'Cancelled'),
            (Failed = 'Failed'),
            (Complete = 'Complete'),
           ]

  def assert_finished
    update_attributes(finished_at: finished_at || Time.now,
                      success: success.nil? ? false : success,
                      running: false)
  end

  def node_uuids
    nodes.map(&:uuid)
  end

  def self.queue
    self.where('state = ?', Queued).order('priority desc, created_at')
  end

  def queue_position
    i = 0
    Job::queue.each do |j|
      if j[:uuid] == self.uuid
        return i
      end
    end
    nil
  end

  def self.running
    self.where('running = ?', true).
      order('priority desc, created_at')
  end

  def lock locked_by_uuid
    transaction do
      self.reload
      unless self.state == Queued and self.is_locked_by_uuid.nil?
        raise AlreadyLockedError
      end
      self.state = Running
      self.is_locked_by_uuid = locked_by_uuid
      self.save!
    end
  end

  protected

  def foreign_key_attributes
    super + %w(output log)
  end

  def skip_uuid_read_permission_check
    super + %w(cancelled_by_client_uuid)
  end

  def skip_uuid_existence_check
    super + %w(output log)
  end

  def set_priority
    if self.priority.nil?
      self.priority = 0
    end
    true
  end

  def ensure_script_version_is_commit
    if self.state == Running
      # Apparently client has already decided to go for it. This is
      # needed to run a local job using a local working directory
      # instead of a commit-ish.
      return true
    end
    if new_record? or script_version_changed?
      sha1 = Commit.find_commit_range(current_user, self.repository, nil, self.script_version, nil)[0] rescue nil
      if sha1
        self.supplied_script_version = self.script_version if self.supplied_script_version.nil? or self.supplied_script_version.empty?
        self.script_version = sha1
      else
        self.errors.add :script_version, "#{self.script_version} does not resolve to a commit"
        return false
      end
    end
  end

  def ensure_unique_submit_id
    if !submit_id.nil?
      if Job.where('submit_id=?',self.submit_id).first
        raise SubmitIdReused.new
      end
    end
    true
  end

  def find_docker_image_locator
    # Find the Collection that holds the Docker image specified in the
    # runtime constraints, and store its locator in docker_image_locator.
    unless runtime_constraints.is_a? Hash
      # We're still in validation stage, so we can't assume
      # runtime_constraints isn't something horrible like an array or
      # a string. Treat those cases as "no docker image supplied";
      # other validations will fail anyway.
      self.docker_image_locator = nil
      return true
    end
    image_search = runtime_constraints['docker_image']
    image_tag = runtime_constraints['docker_image_tag']
    if image_search.nil?
      self.docker_image_locator = nil
      true
    elsif coll = Collection.for_latest_docker_image(image_search, image_tag)
      self.docker_image_locator = coll.portable_data_hash
      true
    else
      errors.add(:docker_image_locator, "not found for #{image_search}")
      false
    end
  end

  def dependencies
    deps = {}
    queue = self.script_parameters.values
    while not queue.empty?
      queue = queue.flatten.compact.collect do |v|
        if v.is_a? Hash
          v.values
        elsif v.is_a? String
          v.match(/^(([0-9a-f]{32})\b(\+[^,]+)?,?)*$/) do |locator|
            deps[locator.to_s] = true
          end
          nil
        end
      end
    end
    deps.keys
  end

  def permission_to_update
    if is_locked_by_uuid_was and !(current_user and
                                   (current_user.uuid == is_locked_by_uuid_was or
                                    current_user.uuid == system_user.uuid))
      if script_changed? or
          script_parameters_changed? or
          script_version_changed? or
          (!cancelled_at_was.nil? and
           (cancelled_by_client_uuid_changed? or
            cancelled_by_user_uuid_changed? or
            cancelled_at_changed?)) or
          started_at_changed? or
          finished_at_changed? or
          running_changed? or
          success_changed? or
          output_changed? or
          log_changed? or
          tasks_summary_changed? or
          state_changed?
        logger.warn "User #{current_user.uuid if current_user} tried to change protected job attributes on locked #{self.class.to_s} #{uuid_was}"
        return false
      end
    end
    if !is_locked_by_uuid_changed?
      super
    else
      if !current_user
        logger.warn "Anonymous user tried to change lock on #{self.class.to_s} #{uuid_was}"
        false
      elsif is_locked_by_uuid_was and is_locked_by_uuid_was != current_user.uuid
        logger.warn "User #{current_user.uuid} tried to steal lock on #{self.class.to_s} #{uuid_was} from #{is_locked_by_uuid_was}"
        false
      elsif !is_locked_by_uuid.nil? and is_locked_by_uuid != current_user.uuid
        logger.warn "User #{current_user.uuid} tried to lock #{self.class.to_s} #{uuid_was} with uuid #{is_locked_by_uuid}"
        false
      else
        super
      end
    end
  end

  def update_modified_by_fields
    if self.cancelled_at_changed?
      # Ensure cancelled_at cannot be set to arbitrary non-now times,
      # or changed once it is set.
      if self.cancelled_at and not self.cancelled_at_was
        self.cancelled_at = Time.now
        self.cancelled_by_user_uuid = current_user.uuid
        self.cancelled_by_client_uuid = current_api_client.andand.uuid
        @need_crunch_dispatch_trigger = true
      else
        self.cancelled_at = self.cancelled_at_was
        self.cancelled_by_user_uuid = self.cancelled_by_user_uuid_was
        self.cancelled_by_client_uuid = self.cancelled_by_client_uuid_was
      end
    end
    super
  end

  def trigger_crunch_dispatch_if_cancelled
    if @need_crunch_dispatch_trigger
      File.open(Rails.configuration.crunch_refresh_trigger, 'wb') do
        # That's all, just create/touch a file for crunch-job to see.
      end
    end
  end

  def update_timestamps_when_state_changes
    return if not (state_changed? or new_record?)
    case state
    when Running
      self.started_at ||= Time.now
    when Failed, Complete
      self.finished_at ||= Time.now
    when Cancelled
      self.cancelled_at ||= Time.now
    end

    # TODO: Remove the following case block when old "success" and
    # "running" attrs go away. Until then, this ensures we still
    # expose correct success/running flags to older clients, even if
    # some new clients are writing only the new state attribute.
    case state
    when Queued
      self.running = false
      self.success = nil
    when Running
      self.running = true
      self.success = nil
    when Cancelled, Failed
      self.running = false
      self.success = false
    when Complete
      self.running = false
      self.success = true
    end
    self.running ||= false # Default to false instead of nil.

    true
  end

  def update_state_from_old_state_attrs
    # If a client has touched the legacy state attrs, update the
    # "state" attr to agree with the updated values of the legacy
    # attrs.
    #
    # TODO: Remove this method when old "success" and "running" attrs
    # go away.
    if cancelled_at_changed? or
        success_changed? or
        running_changed? or
        state.nil?
      if cancelled_at
        self.state = Cancelled
      elsif success == false
        self.state = Failed
      elsif success == true
        self.state = Complete
      elsif running == true
        self.state = Running
      else
        self.state = Queued
      end
    end
    true
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
             # state isn't set yet
             true
           when Queued
             # Permit going from queued to any state
             true
           when Running
             # From running, may only transition to a finished state
             [Complete, Failed, Cancelled].include? self.state
           when Complete, Failed, Cancelled
             # Once in a finished state, don't permit any more state changes
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
