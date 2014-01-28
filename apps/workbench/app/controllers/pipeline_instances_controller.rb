class PipelineInstancesController < ApplicationController
  skip_before_filter :find_object_by_uuid, only: :compare
  before_filter :find_objects_by_uuid, only: :compare

  def compare
  end

  protected
  def find_objects_by_uuid
    @objects = model_class.where(uuid: params[:uuid])
  end
end
