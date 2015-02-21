class SearchController < ApplicationController
  def find_objects_for_index
    search_what = Group
    if params[:project_uuid]
      # Special case for "search all things in project":
      @filters = @filters.select do |attr, operator, operand|
        not (attr == 'owner_uuid' and operator == '=')
      end
      # Special case for project_uuid is a user uuid:
      if ArvadosBase::resource_class_for_uuid(params[:project_uuid]) == User
        search_what = User.find params[:project_uuid]
      else
        search_what = Group.find params[:project_uuid]
      end
    end
    @objects = search_what.contents(limit: @limit,
                                    offset: @offset,
                                    filters: @filters)
    super
  end

  def next_page_href with_params={}
    super with_params.merge(last_object_class: @objects.last.class.to_s,
                            project_uuid: params[:project_uuid],
                            filters: @filters.to_json)
  end
end
