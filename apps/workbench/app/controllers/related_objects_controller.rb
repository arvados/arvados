class RelatedObjectsController < ApplicationController

  def model_class
    Collection
  end

  def index
    @related_objs = {}
    if params[:q].andand.length.andand > 0
      [Collection, Group, Human, Specimen, Trait].each do |obj|
        @related_objs[obj.name] = obj.where(any: ['contains', params[:q]])
      end
    end
  end
end
