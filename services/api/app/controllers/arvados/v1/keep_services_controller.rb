class Arvados::V1::KeepServicesController < ApplicationController

  def find_objects_for_index
    # all users can list all keep disks
    @objects = model_class.where('1=1')
    super
  end

end
