class Job < ArvadosModel
  include AssignUuid
  include KindAndEtag
  include CommonApiTemplate
  serialize :script_parameters, Hash
  serialize :runtime_constraints, Hash
  serialize :tasks_summary, Hash
  before_create :ensure_unique_submit_id
  before_create :ensure_script_version_is_commit
  before_update :ensure_script_version_is_commit
  after_commit :trigger_crunch_dispatch_if_cancelled, :on => :update

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
    t.add :is_locked_by_uuid
    t.add :log
    t.add :runtime_constraints
    t.add :tasks_summary
    t.add :dependencies
    t.add :log_stream_href
  end

  def assert_finished
    update_attributes(finished_at: finished_at || Time.now,
                      success: success.nil? ? false : success,
                      running: false)
  end

  def log_stream_href
    unless self.finished_at
      "#{current_api_base}/#{self.class.to_s.pluralize.underscore}/#{self.uuid}/log_tail_follow"
    end
  end

  def self.queue
    self.where('started_at is ? and is_locked_by_uuid is ? and cancelled_at is ?',
               nil, nil, nil).
      order('priority desc, created_at')
  end

  protected

  def ensure_script_version_is_commit
    if self.is_locked_by_uuid and self.started_at
      # Apparently client has already decided to go for it. This is
      # needed to run a local job using a local working directory
      # instead of a commit-ish.
      return true
    end
    if new_record? or script_version_changed?
      sha1 = Commit.find_by_commit_ish(self.script_version) rescue nil
      if sha1
        self.script_version = sha1
      else
        raise ArgumentError.new("Specified script_version does not resolve to a commit")
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
                                   current_user.uuid == is_locked_by_uuid_was)
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
        self.cancelled_by_client_uuid = current_api_client.uuid
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

end
