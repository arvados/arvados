class Job < OrvosModel
  include AssignUuid
  include KindAndEtag
  include CommonApiTemplate
  serialize :script_parameters, Hash
  serialize :resource_limits, Hash
  serialize :tasks_summary, Hash
  before_create :ensure_unique_submit_id
  before_create :ensure_script_version_is_commit

  has_many :commit_ancestors, :foreign_key => :descendant, :primary_key => :script_version

  class SubmitIdReused < StandardError
  end

  api_accessible :superuser, :extend => :common do |t|
    t.add :submit_id
    t.add :priority
    t.add :script
    t.add :script_parameters
    t.add :script_version
    t.add :cancelled_at
    t.add :cancelled_by_client
    t.add :cancelled_by_user
    t.add :started_at
    t.add :finished_at
    t.add :output
    t.add :success
    t.add :running
    t.add :is_locked_by
    t.add :log
    t.add :resource_limits
    t.add :tasks_summary
    t.add :dependencies
  end

  def assert_finished
    update_attributes(finished_at: finished_at || Time.now,
                      success: success.nil? ? false : success,
                      running: false)
  end

  protected

  def ensure_script_version_is_commit
    sha1 = Commit.find_by_commit_ish(self.script_version) rescue nil
    if sha1
      self.script_version = sha1
    else
      raise ArgumentError.new("Specified script_version does not resolve to a commit")
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
    self.script_parameters.values.each do |v|
      next unless v.is_a? String
      v.match(/^(([0-9a-f]{32})\b(\+[^,]+)?,?)*$/) do |locator|
        bare_locator = locator[0].gsub(/\+[^,]+/,'')
        deps[bare_locator] = true
      end
    end
    deps.keys
  end

  def permission_to_update
    if is_locked_by_was and !(current_user and
                              current_user.uuid == is_locked_by_was)
      if script_changed? or
          script_parameters_changed? or
          script_version_changed? or
          cancelled_by_client_changed? or
          cancelled_by_user_changed? or
          cancelled_at_changed? or
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
    if !is_locked_by_changed?
      super
    else
      if !current_user
        logger.warn "Anonymous user tried to change lock on #{self.class.to_s} #{uuid_was}"
        false
      elsif is_locked_by_was and is_locked_by_was != current_user.uuid
        logger.warn "User #{current_user.uuid} tried to steal lock on #{self.class.to_s} #{uuid_was} from #{is_locked_by_was}"
        false
      elsif !is_locked_by.nil? and is_locked_by != current_user.uuid
        logger.warn "User #{current_user.uuid} tried to lock #{self.class.to_s} #{uuid_was} with uuid #{is_locked_by}"
        false
      else
        super
      end
    end
  end
end
