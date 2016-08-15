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
    rescue
      errors.add :validate_workflow, "#{self.workflow} is not valid yaml"
    end
  end

  def set_name_and_description
    begin
      old_wf = {}
      old_wf = YAML.load self.workflow_was if !self.workflow_was.nil?
      ['name', 'description'].each do |a|
        if !self.changes.include?(a)
          v = self.read_attribute(a)
          if !v.present? or v == old_wf[a]
            val = @workflow_yaml[a] if self.workflow and @workflow_yaml
            self[a] = val
          end
        end
      end
    rescue ActiveRecord::RecordInvalid
      errors.add :set_name_and_description, "#{self.workflow_was} is not valid yaml"
    end
  end
end
