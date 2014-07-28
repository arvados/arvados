class ProjectsController < ApplicationController
  def model_class
    Group
  end

  def index_pane_list
    %w(Projects)
  end

  def show_pane_list
    if @user_is_manager
      %w(Data_collections Jobs_and_pipelines Pipeline_templates Subprojects Other_objects Sharing Advanced)
    else
      %w(Data_collections Jobs_and_pipelines Pipeline_templates Subprojects Other_objects Advanced)
    end
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

  def move_items
    target_uuid = params['target']
    uuids_to_add = session[:selected_move_items]

    uuids_to_add.
      collect { |x| ArvadosBase::resource_class_for_uuid(x) }.
      uniq.
      each do |resource_class|
      resource_class.filter([['uuid','in',uuids_to_add]]).each do |dst|
        if resource_class == Collection
          dst = Link.new(owner_uuid: target_object.uuid,
                         tail_uuid: target_object.uuid,
                         head_uuid: target_uuid,
                         link_class: 'name',
                         name: target_uuid)
        else
          dst.owner_uuid = target_uuid
          dst.tail_uuid = target_uuid if dst.class == Link
        end
        begin
          dst.save!
        rescue
          dst.name += " (#{Time.now.localtime})" if dst.respond_to? :name=
          dst.save!
        end
      end
    end
    session[:selected_move_items] = nil
    redirect_to controller: 'projects', action: :show, id: target_uuid
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
    if ArvadosBase::resource_class_for_uuid(@object.owner_uuid) == Group
      params[:return_to] ||= group_path(@object.owner_uuid)
    else
      params[:return_to] ||= projects_path
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
                                filters: params[:filters],
                                offset: params[:offset] || 0)
    @logs = Log.limit(10).filter([['object_uuid', '=', @object.uuid]])
    @users = User.limit(10000).
      select(["uuid", "is_active", "first_name", "last_name"]).
      filter([['is_active', '=', 'true']])
    @groups = Group.limit(10000).
      select(["uuid", "name", "description"])

    begin
      @share_links = Link.permissions_for(@object)
      @user_is_manager = true
    rescue ArvadosApiClient::AccessForbiddenException,
           ArvadosApiClient::NotFoundException
      @share_links = []
      @user_is_manager = false
    end

    @objects_and_names = get_objects_and_names @objects

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
                             url_for(offset: next_page_offset, filters: params[:filters], partial: true))
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

  helper_method :get_objects_and_names
  def get_objects_and_names(objects)
    objects_and_names = []
    objects.each do |object|
      if !(name_links = objects.links_for(object, 'name')).empty?
        name_links.each do |name_link|
          objects_and_names << [object, name_link]
        end
      elsif object.respond_to? :name
        objects_and_names << [object, object]
      else
        objects_and_names << [object,
                               Link.new(owner_uuid: @object.uuid,
                                        tail_uuid: @object.uuid,
                                        head_uuid: object.uuid,
                                        link_class: "name",
                                        name: "")]
      end
    end
    objects_and_names
  end

  def share_with
    if not params[:uuids].andand.any?
      @errors = ["No user/group UUIDs specified to share with."]
      return render_error(status: 422)
    end
    results = {"success" => [], "errors" => []}
    params[:uuids].each do |shared_uuid|
      begin
        Link.create(tail_uuid: shared_uuid, link_class: "permission",
                    name: "can_read", head_uuid: @object.uuid)
      rescue ArvadosApiClient::ApiError => error
        error_list = error.api_response.andand[:errors]
        if error_list.andand.any?
          results["errors"] += error_list.map { |e| "#{shared_uuid}: #{e}" }
        else
          error_code = error.api_status || "Bad status"
          results["errors"] << "#{shared_uuid}: #{error_code} response"
        end
      else
        results["success"] << shared_uuid
      end
    end
    if results["errors"].empty?
      results.delete("errors")
      status = 200
    else
      status = 422
    end
    respond_to do |f|
      f.json { render(json: results, status: status) }
    end
  end
end
