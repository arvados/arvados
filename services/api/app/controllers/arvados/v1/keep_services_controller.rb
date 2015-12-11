class Arvados::V1::KeepServicesController < ApplicationController

  skip_before_filter :find_object_by_uuid, only: :accessible
  skip_before_filter :render_404_if_no_object, only: :accessible

  def find_objects_for_index
    # all users can list all keep services
    @objects = model_class.where('1=1')
    super
  end

  def accessible
    if request.headers['X-External-Client'] == '1'
      @objects = model_class.where('service_type=?', 'proxy')
    else
      @objects = model_class.where(model_class.arel_table[:service_type].not_eq('proxy'))
    end
    render_list
  end

end
