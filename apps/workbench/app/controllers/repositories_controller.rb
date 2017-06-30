# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

class RepositoriesController < ApplicationController
  before_filter :set_share_links, if: -> { defined? @object }

  def index_pane_list
    %w(repositories help)
  end

  def show_pane_list
    if @user_is_manager
      panes = super | %w(Sharing)
      panes.insert(panes.length-1, panes.delete_at(panes.index('Advanced'))) if panes.index('Advanced')
      panes
    else
      panes = super
    end
    panes.delete('Attributes') if !current_user.is_admin
    panes
  end

  def show_tree
    @commit = params[:commit]
    @path = params[:path] || ''
    @subtree = @object.ls_subtree @commit, @path.chomp('/')
  end

  def show_blob
    @commit = params[:commit]
    @path = params[:path]
    @blobdata = @object.cat_file @commit, @path
  end

  def show_commit
    @commit = params[:commit]
  end

  def all_repos
    limit = params[:limit].andand.to_i || 100
    offset = params[:offset].andand.to_i || 0
    @filters = params[:filters] || []

    if @filters.any?
      owner_filter = @filters.select do |attr, op, val|
        (attr == 'owner_uuid')
      end
    end

    if !owner_filter.andand.any?
      filters = @filters + [["owner_uuid", "=", current_user.uuid]]
      my_repos = Repository.all.order("name ASC").limit(limit).offset(offset).filter(filters).results
    else      # done fetching all owned repositories
      my_repos = []
    end

    if !owner_filter.andand.any?  # if this is next page request, the first page was still fetching "own" repos
      @filters = @filters.reject do |attr, op, val|
        (attr == 'owner_uuid') or
        (attr == 'name') or
        (attr == 'uuid')
      end
    end

    filters = @filters + [["owner_uuid", "!=", current_user.uuid]]
    other_repos = Repository.all.order("name ASC").limit(limit).offset(offset).filter(filters).results

    @objects = (my_repos + other_repos).first(limit)
  end

  def find_objects_for_index
    return if !params[:partial]

    all_repos

    if @objects.any?
      @next_page_filters = next_page_filters('>=')
      @next_page_href = url_for(partial: :repositories_rows,
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
      (attr == 'owner_uuid') or
      (attr == 'name' and op == nextpage_operator) or
      (attr == 'uuid' and op == 'not in')
    end

    if @objects.any?
      last_obj = @objects.last
      next_page_filters += [['name', nextpage_operator, last_obj.name]]
      next_page_filters += [['uuid', 'not in', [last_obj.uuid]]]
      # if not-owned, it means we are done with owned repos and fetching other repos
      next_page_filters += [['owner_uuid', '!=', last_obj.uuid]] if last_obj.owner_uuid != current_user.uuid
    end

    next_page_filters
  end
end
