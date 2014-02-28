class ActionsController < ApplicationController
  def combine_selected_files_into_collection
    lst = []
    params["selection"].each do |s|
      m = CollectionsHelper.match(s)
      if m
        lst.append(m[1] + m[2])
      end
    end

    collections = Collection.where(uuid: lst)
    
    collections.each do |c| 
      puts c.manifest_text
    end

    '/'
  end

  def post
    if params["combine_selected_files_into_collection"]
      redirect_to combine_selected_files_into_collection
    else
      redirect_to :back
    end
  end
end
