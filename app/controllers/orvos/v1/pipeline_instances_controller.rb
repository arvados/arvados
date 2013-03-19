class Orvos::V1::PipelineInstancesController < ApplicationController
  accept_attribute_as_json :components, Hash
  accept_attribute_as_json :properties, Hash
end
