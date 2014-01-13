class Arvados::V1::CollectionsController < ApplicationController
  def create
    # It's not an error for a client to re-register a manifest that we
    # already know about.
    @object = model_class.new resource_attrs
    begin
      @object.save!
    rescue ActiveRecord::RecordNotUnique
      logger.debug resource_attrs.inspect
      if resource_attrs[:manifest_text] and resource_attrs[:uuid]
        @existing_object = model_class.
          where('uuid=? and manifest_text=?',
                resource_attrs[:uuid],
                resource_attrs[:manifest_text]).
          first
        @object = @existing_object || @object
      end
    end
    show
  end
end
