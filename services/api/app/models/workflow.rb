class Workflow < ArvadosModel
  include HasUuid
  include KindAndEtag
  include CommonApiTemplate

  validate :validate_workflow
  before_save :set_name_and_description

  api_accessible :user, extend: :common do |t|
    t.add :name
    t.add :description
    t.add :workflow
  end

  def validate_workflow
    begin
      @workflow_yaml = YAML.load self.workflow if !workflow.nil?
    rescue => e
      errors.add :workflow, "is not valid yaml: #{e.message}"
    end
  end

  def set_name_and_description
    old_wf = {}
    begin
      old_wf = YAML.load self.workflow_was if !self.workflow_was.nil?
    rescue => e
      logger.warn "set_name_and_description error: #{e.message}"
      return
    end

    ['name', 'description'].each do |a|
      if !self.changes.include?(a)
        v = self.read_attribute(a)
        if !v.present? or v == old_wf[a]
          val = @workflow_yaml[a] if self.workflow and @workflow_yaml
          self[a] = val
        end
      end
    end
  end
end
