class GroupsController < ApplicationController
  def index
    @groups = Group.filter [['group_class', 'not in', ['folder']]]
    @group_uuids = @groups.collect &:uuid
    @links_from = Link.where link_class: 'permission', tail_uuid: @group_uuids
    @links_to = Link.where link_class: 'permission', head_uuid: @group_uuids
    super
  end

  def show
    return redirect_to(folder_path(@object)) if @object.group_class == 'folder'
    super
  end
end
