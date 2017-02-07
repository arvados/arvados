class Arvados::V1::PipelineInstancesController < ApplicationController
  accept_attribute_as_json :components, Hash
  accept_attribute_as_json :properties, Hash
  accept_attribute_as_json :components_summary, Hash

  def cancel
    reload_object_before_update
    @object.cancel params[:cascade]
    show
  end
end
