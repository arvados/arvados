class PipelineTemplatesController < ApplicationController
  
  def show
    @objects = PipelineInstance.where(pipeline_template_uuid: @object.uuid)
    super
  end

  def show_pane_list
    %w(Components Pipelines Attributes Metadata JSON API)
  end

end
