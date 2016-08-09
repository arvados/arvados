class Workflow < ArvadosModel
  include HasUuid
  include KindAndEtag
  include CommonApiTemplate

  validate :validate_workflow
  after_save :set_name_and_description

  api_accessible :user, extend: :common do |t|
    t.add :name
    t.add :description
    t.add :workflow
  end

  def validate_workflow
    begin
      @workflow_yaml = YAML.load self.workflow if !workflow.blank?
    rescue
      errors.add :validate_workflow, "#{self.workflow} is not valid yaml"
    end
  end

  def set_name_and_description
    begin
      old_wf = []
      old_wf = YAML.load self.workflow_was if !self.workflow_was.blank?
      changes = self.changes
      need_save = false
      ['name', 'description'].each do |a|
        if !changes.include?(a)
          v = self.read_attribute(a)
          if !v.present? or v == old_wf[a]
            self[a] = @workflow_yaml[a]
          end
        end
      end
    rescue => e
      errors.add :set_name_and_description, "#{e.message}"
    end
  end
end
