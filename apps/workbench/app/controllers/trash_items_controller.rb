class TrashItemsController < ApplicationController
  def model_class
    Collection
  end

  def find_objects_for_index
    # If it's not the index rows partial display, just return
    # The /index request will again be invoked to display the
    # partial at which time, we will be using the objects found.
    return if !params[:partial]

    trashed_items

    if @objects.any?
      @next_page_filters = next_page_filters('<=')
      @next_page_href = url_for(partial: :trash_rows,
                                filters: @next_page_filters.to_json)
      preload_links_for_objects(@objects.to_a)
    else
      @next_page_href = nil
    end
  end

  def next_page_href with_params={}
    @next_page_href
  end

  def trashed_items
    # API server index doesn't return manifest_text by default, but our
    # callers want it unless otherwise specified.
    @select ||= Collection.columns.map(&:name)
    limit = if params[:limit] then params[:limit].to_i else 100 end
    offset = if params[:offset] then params[:offset].to_i else 0 end

    base_search = Collection.select(@select).include_trash(true).where(is_trashed: true)
    base_search = base_search.filter(params[:filters]) if params[:filters]

    if params[:search].andand.length.andand > 0
      tags = Link.where(any: ['contains', params[:search]])
      @objects = (base_search.limit(limit).offset(offset).where(uuid: tags.collect(&:head_uuid)) |
                      base_search.where(any: ['contains', params[:search]])).
        uniq { |c| c.uuid }
    else
      @objects = base_search.limit(limit).offset(offset)
    end

    @links = Link.where(head_uuid: @objects.collect(&:uuid))
    @collection_info = {}
    @objects.each do |c|
      @collection_info[c.uuid] = {
        tag_links: [],
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
        info[:tag_links] << link
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

  def untrash_items
    @untrashed_uuids = []

    updates = {trash_at: nil}

    params[:selection].collect { |uuid| ArvadosBase.find uuid }.each do |item|
      item.update_attributes updates
      @untrashed_uuids << item.uuid
    end

    respond_to do |format|
      format.js
    end
  end
end
