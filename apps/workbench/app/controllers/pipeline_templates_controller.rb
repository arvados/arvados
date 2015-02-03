class PipelineTemplatesController < ApplicationController
  if Rails.configuration.anonymous_user_token
    skip_around_filter :require_thread_api_token, only: :show
  end

  include PipelineComponentsHelper

  def show
    @objects = PipelineInstance.where(pipeline_template_uuid: @object.uuid)
    super
  end

  def show_pane_list
    %w(Components Pipelines Advanced)
  end
end
