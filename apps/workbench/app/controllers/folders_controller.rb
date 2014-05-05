class FoldersController < ApplicationController
  def model_class
    Group
  end

  def index_pane_list
    %w(My_folders Shared_with_me)
  end

  def remove_item
    raise "Not implemented yet!"
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
    @object
  end

  def show
    @objects = @object.contents include_linked: true
    @share_links = Link.filter([['head_uuid', '=', @object.uuid],
                                ['link_class', '=', 'permission']])
    @logs = Log.limit(10).filter([['object_uuid', '=', @object.uuid]])

    @objects_and_names = []
    @objects.each do |object|
      if !(name_links = @objects.links_for(object, 'name')).empty?
        name_links.each do |name_link|
          @objects_and_names << [object, name_link]
        end
      else
        @objects_and_names << [object,
                               Link.new(tail_uuid: @object.uuid,
                                        head_uuid: object.uuid,
                                        link_class: "name",
                                        name: "")]
      end
    end

    super
  end

  def create
    @new_resource_attrs = (params['folder'] || {}).merge(group_class: 'folder')
    @new_resource_attrs[:name] ||= 'New folder'
    super
  end
end
