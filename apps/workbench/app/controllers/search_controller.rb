class SearchController < ApplicationController
  def find_objects_for_index
    @objects = Group.contents(limit: @limit, offset: @offset, filters: @filters)
    super
  end

  def next_page_href with_params={}
    super with_params.merge(last_object_class: @objects.last.class.to_s)
  end
end
