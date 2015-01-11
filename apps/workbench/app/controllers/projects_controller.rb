class ProjectsController < ApplicationController
  before_filter :set_share_links, if: -> { defined? @object }

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

  def set_share_links
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
  end

  def index_pane_list
    %w(Projects)
  end

  # Returning an array of hashes instead of an array of strings will allow
  # us to tell the interface to get counts for each pane (using :filters).
  # It also seems to me that something like these could be used to configure the contents of the panes.
  def show_pane_list
    pane_list = [
      {
        :name => 'Data_collections',
        :filters => [%w(uuid is_a arvados#collection)]
      },
      {
        :name => 'Jobs_and_pipelines',
        :filters => [%w(uuid is_a) + [%w(arvados#job arvados#pipelineInstance)]]
      },
      {
        :name => 'Pipeline_templates',
        :filters => [%w(uuid is_a arvados#pipelineTemplate)]
      },
      {
        :name => 'Subprojects',
        :filters => [%w(uuid is_a arvados#group)]
      },
      { :name => 'Other_objects',
        :filters => [%w(uuid is_a) + [%w(arvados#human arvados#specimen arvados#trait)]]
      }
    ]
    pane_list << { :name => 'Sharing',
                   :count => @share_links.count } if @user_is_manager
    pane_list << { :name => 'Advanced' }
  end

  # Called via AJAX and returns Javascript that populates tab counts into tab titles.
  # References #show_pane_list action which should return an array of hashes each with :name
  # and then optionally a :filters to run or a straight up :count
  #
  # This action could easily be moved to the ApplicationController to genericize the tab_counts behaviour,
  # but one or more new routes would have to be created, the js.erb would also have to be moved
  def tab_counts
    @tab_counts = {}
    show_pane_list.each do |pane|
      if pane.is_a?(Hash)
        if pane[:count]
          @tab_counts[pane[:name]] = pane[:count]
        elsif pane[:filters]
          @tab_counts[pane[:name]] = @object.contents(filters: pane[:filters]).items_available
        end
      end
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
        begin
          item.update_attributes owner_uuid: current_user.uuid
          @removed_uuids << item.uuid
        rescue ArvadosApiClient::ApiErrorResponseException => e
          if e.message.include? 'collection_owner_uuid_name_unique'
            rename_to = item.name + ' removed from ' +
                        (@object.name ? @object.name : @object.uuid) +
                        ' at ' + Time.now.to_s
            updates = {}
            updates[:name] = rename_to
            updates[:owner_uuid] = current_user.uuid
            item.update_attributes updates
            @removed_uuids << item.uuid
          else
            raise
          end
        end
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

      # To avoid losing items that have the same created_at as the
      # last item on this page, we retrieve an overlapping page with a
      # "created_at <= last_created_at" filter, then remove duplicates
      # with a "uuid not in [...]" filter (see below).
      nextpage_operator = /\bdesc$/i =~ @order[0] ? '<=' : '>='

      @objects = []
      @name_link_for = {}
      kind_filters.each do |attr,op,val|
        (val.is_a?(Array) ? val : [val]).each do |type|
          objects = @object.contents(order: @order,
                                     limit: @limit,
                                     include_linked: true,
                                     filters: (@filters - kind_filters + [['uuid', 'is_a', type]]),
                                    )
          objects.each do |object|
            @name_link_for[object.andand.uuid] = objects.links_for(object, 'name').first
          end
          @objects += objects
        end
      end
      @objects = @objects.to_a.sort_by(&:created_at)
      @objects.reverse! if nextpage_operator == '<='
      @objects = @objects[0..@limit-1]
      @next_page_filters = @filters.reject do |attr,op,val|
        (attr == 'created_at' and op == nextpage_operator) or
          (attr == 'uuid' and op == 'not in')
      end

      if @objects.any?
        last_created_at = @objects.last.created_at

        last_uuids = []
        @objects.each do |obj|
          last_uuids << obj.uuid if obj.created_at.eql?(last_created_at)
        end

        @next_page_filters += [['created_at',
                                nextpage_operator,
                                last_created_at]]
        @next_page_filters += [['uuid', 'not in', last_uuids]]
        @next_page_href = url_for(partial: :contents_rows,
                                  limit: @limit,
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
      @next_page_href = next_page_href(partial: :contents_rows,
                                       filters: @filters.to_json,
                                       order: @order.to_json)
    end

    preload_links_for_objects(@objects.to_a)
  end

  def show
    if !@object
      return render_not_found("object not found")
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
