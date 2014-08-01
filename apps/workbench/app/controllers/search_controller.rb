class SearchController < ApplicationController
  def find_objects_for_index
    search_what = Group
    if params[:project_uuid]
      # Special case for "search all things in project":
      @filters = @filters.select do |attr, operator, operand|
        not (attr == 'owner_uuid' and operator == '=')
      end
      search_what = Group.find(params[:project_uuid])
    end
    @objects = search_what.contents(limit: @limit,
                                    offset: @offset,
                                    filters: @filters,
                                    include_linked: true)
    super
  end

  def next_page_href with_params={}
    super with_params.merge(last_object_class: @objects.last.class.to_s)
  end
end
