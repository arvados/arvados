class SearchController < ApplicationController
  def find_objects_for_index
    @objects = Group.contents(limit: @limit, offset: @offset, filters: @filters)
    super
  end
end
