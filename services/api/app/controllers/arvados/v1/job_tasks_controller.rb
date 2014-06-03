class Arvados::V1::JobTasksController < ApplicationController
  accept_attribute_as_json :parameters, Hash
end
