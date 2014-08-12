class LinksController < ApplicationController
  def show
    if @object.link_class == 'name' and
        Collection == ArvadosBase::resource_class_for_uuid(@object.head_uuid)
      return redirect_to collection_path(@object.uuid)
    end
    super
  end
end
