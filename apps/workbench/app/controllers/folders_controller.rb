class FoldersController < ApplicationController
  def model_class
    Group
  end

  def index
    @objects = Group.where group_class: 'folder'
  end

  def show
    @objects = @object.contents include_linked: true
    @logs = Log.limit(10).filter([['object_uuid', '=', @object.uuid]])
    super
  end

  def create
    params['folder'] ||= {}.with_indifferent_access
    params['folder']['group_class'] = 'folder'
    super
  end
end
