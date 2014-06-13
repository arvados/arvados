class ProjectsController < ApplicationController
  def model_class
    Group
  end

  def index_pane_list
    %w(Projects)
  end

  def show_pane_list
    %w(Contents Permissions Advanced)
  end

  def remove_item
    params[:item_uuids] = [params[:item_uuid]]
    remove_items
    render template: 'projects/remove_items'
  end

  def remove_items
    @removed_uuids = []
    links = []
    params[:item_uuids].collect { |uuid| ArvadosBase.find uuid }.each do |item|
      if (item.class == Link and
          item.link_class == 'name' and
          item.tail_uuid == @object.uuid)
        # Given uuid is a name link, linking an object to this
        # project. First follow the link to find the item we're removing,
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
        # Object is owned by this project. Remove it from the project by
        # changing owner to the current user.
        item.update_attributes owner_uuid: current_user.uuid
        @removed_uuids << item.uuid
      end
    end
  end

  def destroy
    while (objects = Link.filter([['owner_uuid','=',@object.uuid],
                                  ['tail_uuid','=',@object.uuid]])).any?
      objects.each do |object|
        object.destroy
      end
    end
    while (objects = @object.contents(include_linked: false)).any?
      objects.each do |object|
        object.update_attributes! owner_uuid: current_user.uuid
      end
    end
    super
  end

  def find_objects_for_index
    @objects = all_projects
    super
  end

  def show
    if !@object
      return render_not_found("object not found")
    end
    @objects = @object.contents(limit: 50,
                                include_linked: true,
                                offset: params[:offset] || 0)
    @share_links = Link.filter([['head_uuid', '=', @object.uuid],
                                ['link_class', '=', 'permission']])
    @logs = Log.limit(10).filter([['object_uuid', '=', @object.uuid]])

    @objects_and_names = []
    @objects.each do |object|
      if !(name_links = @objects.links_for(object, 'name')).empty?
        name_links.each do |name_link|
          @objects_and_names << [object, name_link]
        end
      elsif object.respond_to? :name
        @objects_and_names << [object, object]
      else
        @objects_and_names << [object,
                               Link.new(owner_uuid: @object.uuid,
                                        tail_uuid: @object.uuid,
                                        head_uuid: object.uuid,
                                        link_class: "name",
                                        name: "")]
      end
    end
    if params[:partial]
      respond_to do |f|
        f.json {
          render json: {
            content: render_to_string(partial: 'show_contents_rows.html',
                                      formats: [:html],
                                      locals: {
                                        objects_and_names: @objects_and_names,
                                        project: @object
                                      }),
            next_page_href: (next_page_offset and
                             url_for(offset: next_page_offset, partial: true))
          }
        }
      end
    else
      super
    end
  end

  def create
    @new_resource_attrs = (params['project'] || {}).merge(group_class: 'project')
    @new_resource_attrs[:name] ||= 'New project'
    super
  end

  def update
    @updates = params['project']
    super
  end
end
