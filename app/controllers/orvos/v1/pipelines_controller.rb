class Orvos::V1::PipelinesController < ApplicationController
  accept_attribute_as_json :components, Hash
end
