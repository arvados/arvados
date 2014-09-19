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
  validate :ensure_script_version_is_commit
  validate :find_docker_image_locator
  before_validation :verify_status
  before_create :set_state_before_save
  before_save :set_state_before_save

  has_many :commit_ancestors, :foreign_key => :descendant, :primary_key => :script_version

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

  def self.queue
    self.where('started_at is ? and is_locked_by_uuid is ? and cancelled_at is ? and success is ?',
               nil, nil, nil, nil).
      order('priority desc, created_at')
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
    if self.is_locked_by_uuid and self.started_at
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
          tasks_summary_changed?
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

  def verify_status
    changed_attributes = self.changed

    if new_record?
      self.state = Queued
    elsif 'state'.in? changed_attributes
      case self.state
      when Queued
        self.running = false
        self.success = nil
      when Running
        if !self.is_locked_by_uuid
          return false
        end
        if !self.started_at
          self.started_at = Time.now
        end
        self.running = true
        self.success = nil
      when Cancelled
        if !self.cancelled_at
          self.cancelled_at = Time.now
        end
        self.running = false
        self.success = false
      when Failed
        if !self.finished_at
          self.finished_at = Time.now
        end
        self.running = false
        self.success = false
      when Complete
        if !self.finished_at
          self.finished_at = Time.now
        end
        self.running = false
        self.success = true
      end
    elsif 'cancelled_at'.in? changed_attributes
      self.state = Cancelled
      self.running = false
      self.success = false
    elsif 'success'.in? changed_attributes
      if self.cancelled_at
        self.state = Cancelled
        self.running = false
        self.success = false
      else
        if self.success
          self.state = Complete
        else
          self.state = Failed
        end
        if !self.finished_at
          self.finished_at = Time.now
        end
        self.running = false
      end
    elsif 'running'.in? changed_attributes
      if self.running
        self.state = Running
        if !self.started_at
          self.started_at = Time.now
        end
      else
        self.state = nil # let set_state_before_save determine what the state should be
        self.started_at = nil
      end
    end
    true
  end

  def set_state_before_save
    if !self.state
      if self.cancelled_at
        self.state = Cancelled
      elsif self.success
        self.state = Complete
      elsif (!self.success.nil? && !self.success)
        self.state = Failed
      elsif (self.running && self.success.nil? && !self.cancelled_at)
        self.state = Running
      elsif !self.started_at && !self.cancelled_at && !self.is_locked_by_uuid && self.success.nil?
        self.state = Queued
      end
    end
 
    if self.state.in?(States)
      true
    else
      errors.add :state, "'#{state.inspect} must be one of: [#{States.join ', '}]"
      false
    end
  end

end
