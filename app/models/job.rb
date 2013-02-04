class Job < OrvosModel
  include AssignUuid
  include KindAndEtag
  include CommonApiTemplate
  serialize :command_parameters, Hash
  before_create :ensure_unique_submit_id

  class SubmitIdReused < StandardError
  end

  api_accessible :superuser, :extend => :common do |t|
    t.add :submit_id
    t.add :priority
    t.add :command
    t.add :command_parameters
    t.add :command_version
    t.add :cancelled_at
    t.add :cancelled_by_client
    t.add :cancelled_by_user
    t.add :started_at
    t.add :finished_at
    t.add :success
    t.add :running
    t.add :dependencies
  end

  protected

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
    self.command_parameters.values.each do |v|
      v.match(/^(([0-9a-f]{32})\b(\+[^,]+)?,?)*$/) do |locator|
        bare_locator = locator[0].gsub(/\+[^,]+/,'')
        deps[bare_locator] = true
      end
    end
    deps.keys
  end
end
