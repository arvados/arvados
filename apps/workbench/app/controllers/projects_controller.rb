class ProjectsController < ApplicationController
  def model_class
    Group
  end

  def find_object_by_uuid
    if current_user and params[:uuid] == current_user.uuid
      @object = current_user.dup
      @object.uuid = current_user.uuid
      class << @object
        def name
          'Home'
        end
        def description
          ''
        end
        def attribute_editable? attr, *args
          case attr
          when 'description', 'name'
            false
          else
            super
          end
        end
      end
    else
      super
    end
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

  def load_contents_objects kinds=[]
    kind_filters = @filters.select do |attr,op,val|
      op == 'is_a' and val.is_a? Array and val.count > 1
    end
    if /^created_at\b/ =~ @order[0] and kind_filters.count == 1
      # If filtering on multiple types and sorting by date: Get the
      # first page of each type, sort the entire set, truncate to one
      # page, and use the last item on this page as a filter for
      # retrieving the next page. Ideally the API would do this for
      # us, but it doesn't (yet).
      nextpage_operator = /\bdesc$/i =~ @order[0] ? '<' : '>'
      @objects = []
      @name_link_for = {}
      kind_filters.each do |attr,op,val|
        (val.is_a?(Array) ? val : [val]).each do |type|
          objects = @object.contents(order: @order,
                                     limit: @limit,
                                     include_linked: true,
                                     filters: (@filters - kind_filters + [['uuid', 'is_a', type]]),
                                     offset: @offset)
          objects.each do |object|
            @name_link_for[object.andand.uuid] = objects.links_for(object, 'name').first
          end
          @objects += objects
        end
      end
      @objects = @objects.to_a.sort_by(&:created_at)
      @objects.reverse! if nextpage_operator == '<'
      @objects = @objects[0..@limit-1]
      @next_page_filters = @filters.reject do |attr,op,val|
        attr == 'created_at' and op == nextpage_operator
      end
      if @objects.any?
        @next_page_filters += [['created_at',
                                nextpage_operator,
                                @objects.last.created_at]]
        @next_page_href = url_for(partial: :contents_rows,
                                  filters: @next_page_filters.to_json)
      else
        @next_page_href = nil
      end
    else
      @objects = @object.contents(order: @order,
                                  limit: @limit,
                                  include_linked: true,
                                  filters: @filters,
                                  offset: @offset)
      @next_page_href = next_page_href(partial: :contents_rows)
    end

    preload_links_for_objects(@objects.to_a)
  end

  def show
    if !@object
      return render_not_found("object not found")
    end

    @user_is_manager = false
    @share_links = []
    if @object.uuid != current_user.uuid
      begin
        @share_links = Link.permissions_for(@object)
        @user_is_manager = true
      rescue ArvadosApiClient::AccessForbiddenException,
        ArvadosApiClient::NotFoundException
      end
    end

    if params[:partial]
      load_contents_objects
      respond_to do |f|
        f.json {
          render json: {
            content: render_to_string(partial: 'show_contents_rows.html',
                                      formats: [:html]),
            next_page_href: @next_page_href
          }
        }
      end
    else
      @objects = []
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
  def get_objects_and_names(objects=nil)
    objects = @objects if objects.nil?
    objects_and_names = []
    objects.each do |object|
      if objects.respond_to? :links_for and
          !(name_links = objects.links_for(object, 'name')).empty?
        name_links.each do |name_link|
          objects_and_names << [object, name_link]
        end
      elsif @name_link_for.andand[object.uuid]
        objects_and_names << [object, @name_link_for[object.uuid]]
      elsif object.respond_to? :name
        objects_and_names << [object, object]
      elsif not Collection.attribute_info.include?(:name)
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
