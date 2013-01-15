class Orvos::V1::SchemaController < ApplicationController
  skip_before_filter :find_object_by_uuid
  def show
    Rails.application.eager_load!
    classes = {}
    ActiveRecord::Base.descendants.each do |k|
      classes[k] = k.columns.collect do |col|
        if k.serialized_attributes.has_key? col.name
          { name: col.name,
            type: k.serialized_attributes[col.name].object_class.to_s }
        else
          { name: col.name,
            type: col.type }
        end
      end
    end
    render json: classes
  end
end
