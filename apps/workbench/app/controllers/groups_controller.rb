class GroupsController < ApplicationController
  def index
    @groups = Group.limit(10000).all
    @group_uuids = @groups.collect &:uuid
    @owned_users = User.where owner_uuid: @group_uuids
    @links_from = Link.where link_class: 'permission', tail_uuid: @group_uuids
    @links_to = Link.where link_class: 'permission', head_uuid: @group_uuids
  end
end
