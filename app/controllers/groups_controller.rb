class GroupsController < ApplicationController
  before_filter :ensure_current_user_is_admin

  def index
    @groups = Group.limit(10000).all
    @group_uuids = @groups.collect &:uuid
    @owned_users = User.where owner: @group_uuids
    @links_from = Link.where link_class: 'permission', tail_uuid: @group_uuids
    @links_to = Link.where link_class: 'permission', head_uuid: @group_uuids
  end

  def show
    @collections = Collection.where(owner: @object.uuid)
    @names = {}
    @keep_flag = {}
    Link.
      limit(10000).
      where(head_uuid: @collections.collect(&:uuid)).
      each do |link|
      if link.properties[:name]
        @names[link.head_uuid] ||= []
        @names[link.head_uuid] << link.properties[:name]
      end
      if link.link_class == 'resources' and link.name == 'wants'
        @keep_flag[link.head_uuid] = true
      end
    end
    @collections_total_bytes = @collections.collect(&:total_bytes).inject(0,&:+)
    super
  end
end
