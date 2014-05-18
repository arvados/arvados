class LinksController < ApplicationController
  def show
    if @object and request.method == 'GET' and action_name == 'show'
      if @object.link_class == 'name' and @object.head_uuid
        # Rather than show a name link, show the named object
        klass = ArvadosBase.
          resource_class_for_uuid(@object.head_uuid) rescue nil
        if klass and klass != Link
          return redirect_to(action: :show,
                             controller: klass.to_s.tableize,
                             id: @object.uuid)
        end
      end
    end
    super
  end
end
