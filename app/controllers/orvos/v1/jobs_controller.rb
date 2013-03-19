class Orvos::V1::JobsController < ApplicationController
  accept_attribute_as_json :script_parameters, Hash
  accept_attribute_as_json :resource_limits, Hash
  accept_attribute_as_json :tasks_summary, Hash
end
