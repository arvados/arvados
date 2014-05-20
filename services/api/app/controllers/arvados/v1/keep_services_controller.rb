class Arvados::V1::KeepServicesController < ApplicationController

  def find_objects_for_index
    # all users can list all keep services
    @objects = model_class.where('1=1')
    super
  end

  def accessable
    if request.headers['X-Keep-Proxy-Required']
      @objects = model_class.where('service_type=?', 'proxy')
    else
      @objects = model_class.where('service_type=?', 'disk')
    end

    render_list
  end

end
