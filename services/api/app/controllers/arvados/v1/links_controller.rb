# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

class Arvados::V1::LinksController < ApplicationController

  def check_uuid_kind uuid, kind
    if kind and ArvadosModel::resource_class_for_uuid(uuid).andand.kind != kind
      send_error("'#{kind}' does not match uuid '#{uuid}', expected '#{ArvadosModel::resource_class_for_uuid(uuid).andand.kind}'",
                 status: 422)
      nil
    else
      true
    end
  end

  def create
    return if ! check_uuid_kind resource_attrs[:head_uuid], resource_attrs[:head_kind]
    return if ! check_uuid_kind resource_attrs[:tail_uuid], resource_attrs[:tail_kind]

    resource_attrs.delete :head_kind
    resource_attrs.delete :tail_kind

    if resource_attrs[:link_class] == 'permission' && Link::PermLevel[resource_attrs[:name]]
      existing = Link.
                   lock. # select ... for update
                   where(link_class: 'permission',
                         tail_uuid: resource_attrs[:tail_uuid],
                         head_uuid: resource_attrs[:head_uuid],
                         name: Link::PermLevel.keys).first
      if existing
        @object = existing
        if Link::PermLevel[resource_attrs[:name]] > Link::PermLevel[existing.name]
          # upgrade existing permission link to the requested level.
          return update
        else
          # no-op: existing permission is already greater or equal to
          # the newly requested permission.
          return show
        end
      end
    elsif resource_attrs[:link_class] == 'permission' &&
          resource_attrs[:name] == 'can_login' &&
          resource_attrs[:properties].respond_to?(:has_key?) &&
          resource_attrs[:properties].has_key?(:username)
      existing = Link.
                   lock. # select ... for update
                   where(link_class: 'permission',
                         tail_uuid: resource_attrs[:tail_uuid],
                         head_uuid: resource_attrs[:head_uuid]).
                   where('properties @> ?', SafeJSON.dump({'username' => resource_attrs[:properties][:username]})).
                   first
      if existing
        @object = existing
        return show
      end
    end

    super
  end

  def get_permissions
    if current_user.andand.can?(manage: @object)
      # find all links and return them
      @objects = Link.unscoped.where(link_class: "permission",
                                     head_uuid: params[:uuid])
      @offset = 0
      @limit = @objects.count
      render_list
    else
      render :json => { errors: ['Forbidden'] }.to_json, status: 403
    end
  end

  protected

  def find_object_by_uuid(with_lock: false)
    if params[:id] && params[:id].match(/\D/)
      params[:uuid] = params.delete :id
    end
    if action_name == 'get_permissions'
      # get_permissions accepts a UUID for any kind of object.
      @object = ArvadosModel::resource_class_for_uuid(params[:uuid])
        .readable_by(*@read_users)
        .where(uuid: params[:uuid])
        .first
    elsif !current_user
      super(with_lock: with_lock)
    else
      # The usual permission-filtering index query is unnecessarily
      # inefficient, and doesn't match all permission links that
      # should be visible (see #18865).  Instead, we look up the link
      # by UUID, then check whether (a) its tail_uuid is the current
      # user or (b) its head_uuid is an object the current_user
      # can_manage.
      model = Link
      if with_lock && Rails.configuration.API.LockBeforeUpdate
        model = model.lock
      end
      link = model.unscoped.where(uuid: params[:uuid]).first
      if link && link.link_class != 'permission'
        # Not a permission link. Re-fetch using generic
        # permission-filtering query.
        super(with_lock: with_lock)
      elsif link && (current_user.uuid == link.tail_uuid ||
                     current_user.can?(manage: link.head_uuid))
        # Permission granted.
        @object = link
      else
        # Permission denied, i.e., link is invisible => 404.
        @object = nil
      end
    end
  end

  # Overrides ApplicationController load_where_param
  def load_where_param
    super

    # head_kind and tail_kind columns are now virtual,
    # equivalent functionality is now provided by
    # 'is_a', so fix up any old-style 'where' clauses.
    if @where
      @filters ||= []
      if @where[:head_kind]
        @filters << ['head_uuid', 'is_a', @where[:head_kind]]
        @where.delete :head_kind
      end
      if @where[:tail_kind]
        @filters << ['tail_uuid', 'is_a', @where[:tail_kind]]
        @where.delete :tail_kind
      end
    end
  end

  # Overrides ApplicationController load_filters_param
  def load_filters_param
    super

    # head_kind and tail_kind columns are now virtual,
    # equivalent functionality is now provided by
    # 'is_a', so fix up any old-style 'filter' clauses.
    @filters = @filters.map do |k|
      if k[0] == 'head_kind' and k[1] == '='
        ['head_uuid', 'is_a', k[2]]
      elsif k[0] == 'tail_kind' and k[1] == '='
        ['tail_uuid', 'is_a', k[2]]
      else
        k
      end
    end

    # If the provided filters are enough to limit the results to
    # permission links with specific head_uuids or
    # tail_uuid=current_user, bypass the normal readable_by query
    # (which doesn't match all can_manage-able items, see #18865) --
    # just ensure the current user actually has can_manage permission
    # for the provided head_uuids, removing any that don't. At that
    # point the caller's filters are an effective permission filter.
    if @filters.include?(['link_class', '=', 'permission'])
      @filters.map do |k|
        if k[0] == 'tail_uuid' && k[1] == '=' && k[2] == current_user.uuid
          @objects = Link.unscoped
        elsif k[0] == 'head_uuid'
          if k[1] == '=' && current_user.can?(manage: k[2])
            @objects = Link.unscoped
          elsif k[1] == 'in'
            # Modify the filter operand element (k[2]) in place,
            # removing any non-permitted UUIDs.
            k[2].select! do |head_uuid|
              current_user.can?(manage: head_uuid)
            end
            @objects = Link.unscoped
          end
        end
      end
    end
  end

end
