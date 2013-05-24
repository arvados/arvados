class JobTask < ArvadosModel
  include AssignUuid
  include KindAndEtag
  include CommonApiTemplate
  serialize :parameters, Hash
  after_update :delete_created_job_tasks_if_failed
  after_update :assign_created_job_tasks_qsequence_if_succeeded

  api_accessible :superuser, :extend => :common do |t|
    t.add :job_uuid
    t.add :created_by_job_task
    t.add :sequence
    t.add :qsequence
    t.add :parameters
    t.add :output
    t.add :progress
    t.add :success
  end

  protected

  def delete_created_job_tasks_if_failed
    if self.success == false and self.success != self.success_was
      JobTask.delete_all ['created_by_job_task = ?', self.uuid]
    end
  end

  def assign_created_job_tasks_qsequence_if_succeeded
    if self.success == false and self.success != self.success_was
      # xxx qsequence should be sequential as advertised; for now at
      # least it's non-decreasing.
      JobTask.update_all(['qsequence = ?', (Time.now.to_f*10000000).to_i],
                         ['created_by_job_task = ?', self.uuid])
    end
  end
end
