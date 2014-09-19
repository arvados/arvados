class JobTask < ArvadosModel
  include HasUuid
  include KindAndEtag
  include CommonApiTemplate
  serialize :parameters, Hash
  before_create :set_default_qsequence
  after_update :delete_created_job_tasks_if_failed

  api_accessible :user, extend: :common do |t|
    t.add :job_uuid
    t.add :created_by_job_task_uuid
    t.add :sequence
    t.add :qsequence
    t.add :parameters
    t.add :output
    t.add :progress
    t.add :success
    t.add :started_at
    t.add :finished_at
  end

  protected

  def delete_created_job_tasks_if_failed
    if self.success == false and self.success != self.success_was
      JobTask.delete_all ['created_by_job_task_uuid = ?', self.uuid]
    end
  end

  def set_default_qsequence
    self.qsequence ||= self.class.connection.
      select_value("SELECT nextval('job_tasks_qsequence_seq')")
  end
end
