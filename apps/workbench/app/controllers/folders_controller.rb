class FoldersController < ApplicationController
  def model_class
    Group
  end

  def index_pane_list
    %w(Folders)
  end

  def remove_item
    @removed_uuids = []
    links = []
    item = ArvadosBase.find params[:item_uuid]
    if (item.class == Link and
        item.link_class == 'name' and
        item.tail_uuid = @object.uuid)
      # Given uuid is a name link, linking an object to this
      # folder. First follow the link to find the item we're removing,
      # then delete the link.
      links << item
      item = ArvadosBase.find item.head_uuid
    else
      # Given uuid is an object. Delete all names.
      links += Link.where(tail_uuid: @object.uuid,
                          head_uuid: item.uuid,
                          link_class: 'name')
    end
    links.each do |link|
      @removed_uuids << link.uuid
      link.destroy
    end
    if item.owner_uuid == @object.uuid
      # Object is owned by this folder. Remove it from the folder by
      # changing owner to the current user.
      item.update_attributes owner_uuid: current_user
      @removed_uuids << item.uuid
    end
  end

  def index
    @objects = Group.where(group_class: 'folder').order('name')
    parent_of = {current_user.uuid => 'me'}
    @objects.each do |ob|
      parent_of[ob.uuid] = ob.owner_uuid
    end
    children_of = {false => [], 'me' => [current_user]}
    @objects.each do |ob|
      if ob.owner_uuid != current_user.uuid and
          not parent_of.has_key? ob.owner_uuid
        parent_of[ob.uuid] = false
      end
      children_of[parent_of[ob.uuid]] ||= []
      children_of[parent_of[ob.uuid]] << ob
    end
    buildtree = lambda do |children_of, root_uuid=false|
      tree = {}
      children_of[root_uuid].andand.each do |ob|
        tree[ob] = buildtree.call(children_of, ob.uuid)
      end
      tree
    end
    sorted_paths = lambda do |tree, depth=0|
      paths = []
      tree.keys.sort_by { |ob| ob.friendly_link_name }.each do |ob|
        paths << {object: ob, depth: depth}
        paths += sorted_paths.call tree[ob], depth+1
      end
      paths
    end
    @my_folder_tree = sorted_paths.call buildtree.call(children_of, 'me')
    @shared_folder_tree = sorted_paths.call buildtree.call(children_of, false)
  end

  def choose
    index
    render partial: 'choose'
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

  def update
    @updates = params['folder']
    super
  end
end
