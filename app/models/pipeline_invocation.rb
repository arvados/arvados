class PipelineInvocation < ActiveRecord::Base
  include AssignUuid
  serialize :components, Hash
  belongs_to :pipeline, :foreign_key => :pipeline_uuid, :primary_key => :uuid

  before_validation :bootstrap_components

  protected
  def bootstrap_components
    if pipeline and (!components or components.empty?)
      self.components = pipeline.components
    end
  end
end
