class GroupsController < ApplicationController
  def index
    @groups = Group.all
    @group_uuids = @groups.collect &:uuid
    @links_from = Link.where link_class: 'permission', tail_uuid: @group_uuids
    @links_to = Link.where link_class: 'permission', head_uuid: @group_uuids
  end

  def show
    @objects = @object.owned_items include_linked: true
    super
  end
end
