class RelatedObjectsController < ApplicationController
  def model_class
    Collection
  end
  def index
    if params[:q].andand.length.andand > 0
      tags = Link.where(any: ['contains', params[:q]])
      @collections = (Collection.where(uuid: tags.collect(&:head_uuid)) |
                      Collection.where(any: ['contains', params[:q]])).
        uniq { |c| c.uuid }
    else
      @collections = Collection.limit(100)
    end
    @links = Link.limit(1000).
      where(head_uuid: @collections.collect(&:uuid))
    @collection_info = {}
    @collections.each do |c|
      @collection_info[c.uuid] = {
        tags: [],
        wanted: false,
        wanted_by_me: false,
        provenance: [],
        links: []
      }
    end
    @links.each do |link|
      @collection_info[link.head_uuid] ||= {}
      info = @collection_info[link.head_uuid]
      case link.link_class
      when 'tag'
        info[:tags] << link.name
      when 'resources'
        info[:wanted] = true
        info[:wanted_by_me] ||= link.tail_uuid == current_user.uuid
      when 'provenance'
        info[:provenance] << link.name
      end
      info[:links] << link
    end
    @request_url = request.url
  end
end
