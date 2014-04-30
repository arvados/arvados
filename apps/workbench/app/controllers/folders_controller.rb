class FoldersController < ApplicationController
  def model_class
    Group
  end

  def index_pane_list
    %w(My_folders Shared_with_me)
  end

  def index
    @my_folders = []
    @shared_with_me = []
    @objects = Group.where(group_class: 'folder').order('name')
    owner_of = {}
    moretodo = true
    while moretodo
      moretodo = false
      @objects.each do |folder|
        if !owner_of[folder.uuid]
          moretodo = true
          owner_of[folder.uuid] = folder.owner_uuid
        end
        if owner_of[folder.owner_uuid]
          if owner_of[folder.uuid] != owner_of[folder.owner_uuid]
            owner_of[folder.uuid] = owner_of[folder.owner_uuid]
            moretodo = true
          end
        end
      end
    end
    @objects.each do |folder|
      if owner_of[folder.uuid] == current_user.uuid
        @my_folders << folder
      else
        @shared_with_me << folder
      end
    end
  end

  def show
    @objects = @object.contents include_linked: true
    @logs = Log.limit(10).filter([['object_uuid', '=', @object.uuid]])
    super
  end

  def create
    @new_resource_attrs = (params['folder'] || {}).merge(group_class: 'folder')
    super
  end
end
