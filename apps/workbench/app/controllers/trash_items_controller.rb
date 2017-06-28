# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

class TrashItemsController < ApplicationController
  def model_class
    Collection
  end

  def index_pane_list
    %w(Recent_trash)
  end

  def find_objects_for_index
    # If it's not the index rows partial display, just return
    # The /index request will again be invoked to display the
    # partial at which time, we will be using the objects found.
    return if !params[:partial]

    trashed_items

    if @objects.any?
      @objects = @objects.sort_by { |obj| obj.trash_at }.reverse
      @next_page_filters = next_page_filters('<=')
      @next_page_href = url_for(partial: :trash_rows,
                                filters: @next_page_filters.to_json)
    else
      @next_page_href = nil
    end
  end

  def next_page_href with_params={}
    @next_page_href
  end

  def next_page_filters nextpage_operator
    next_page_filters = @filters.reject do |attr, op, val|
      (attr == 'trash_at' and op == nextpage_operator) or
      (attr == 'uuid' and op == 'not in')
    end

    if @objects.any?
      last_trash_at = @objects.last.trash_at

      last_uuids = []
      @objects.each do |obj|
        last_uuids << obj.uuid if obj.trash_at.eql?(last_trash_at)
      end

      next_page_filters += [['trash_at', nextpage_operator, last_trash_at]]
      next_page_filters += [['uuid', 'not in', last_uuids]]
    end

    next_page_filters
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
      base_search = base_search.limit(limit).offset(offset)
      @objects = (base_search.where(uuid: tags.collect(&:head_uuid)) |
                  base_search.where(any: ['contains', params[:search]])).
                  uniq { |c| c.uuid }
    else
      @objects = base_search.limit(limit).offset(offset)
    end
  end

  def untrash_items
    @untrashed_uuids = []

    updates = {trash_at: nil}

    Collection.include_trash(1).where(uuid: params[:selection]).each do |c|
      c.untrash
      @untrashed_uuids << c.uuid
    end

    respond_to do |format|
      format.js
    end
  end
end
