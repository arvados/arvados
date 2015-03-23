class GroupsController < ApplicationController
  def index
    @groups = Group.filter [['group_class', '!=', 'project']]
    @group_uuids = @groups.collect &:uuid
    @links_from = Link.where link_class: 'permission', tail_uuid: @group_uuids
    @links_to = Link.where link_class: 'permission', head_uuid: @group_uuids
    render_index
  end

  def show
    if @object.group_class == 'project'
      redirect_to(project_path(@object))
    else
      super
    end
  end
end
