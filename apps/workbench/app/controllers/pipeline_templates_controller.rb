class PipelineTemplatesController < ApplicationController
  before_filter :ensure_current_user_is_admin
  def index
    @objects = model_class.all
  end
end
