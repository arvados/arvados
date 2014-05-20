class Arvados::V1::KeepServicesController < ApplicationController

  skip_before_filter :find_object_by_uuid, only: :accessable
  skip_before_filter :render_404_if_no_object, only: :accessable

  def find_objects_for_index
    # all users can list all keep services
    @objects = model_class.where('1=1')
    super
  end

  def accessable
    puts "Hello world"
    if request.headers['X-Keep-Proxy-Required']
      @objects = model_class.where('service_type=?', 'proxy')
    else
      @objects = model_class.where('service_type=?', 'disk')
    end

    puts "Rendering list now"

    render_list
  end

end
