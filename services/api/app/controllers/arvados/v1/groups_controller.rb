# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

require "trashable"

class Arvados::V1::GroupsController < ApplicationController
  include TrashableController

  before_action :load_include_param, only: [:shared, :contents]
  skip_before_action :find_object_by_uuid, only: :shared
  skip_before_action :render_404_if_no_object, only: :shared

  def self._index_requires_parameters
    (super rescue {}).
      merge({
        include_trash: {
          type: 'boolean', required: false, default: false, description: "Include items whose is_trashed attribute is true.",
        },
      })
  end

  def self._show_requires_parameters
    (super rescue {}).
      merge({
        include_trash: {
          type: 'boolean', required: false, default: false, description: "Show group/project even if its is_trashed attribute is true.",
        },
      })
  end

  def self._contents_requires_parameters
    params = _index_requires_parameters.
      merge({
              uuid: {
                type: 'string', required: false, default: '',
              },
              recursive: {
                type: 'boolean', required: false, default: false, description: 'Include contents from child groups recursively.',
              },
              include: {
                type: 'array', required: false, description: 'Include objects referred to by listed fields in "included" response field. Subsets of ["owner_uuid", "container_uuid"] are supported.',
              },
              include_old_versions: {
                type: 'boolean', required: false, default: false, description: 'Include past collection versions.',
              }
            })
    params
  end

  def self._create_requires_parameters
    super.merge(
      {
        async: {
          required: false,
          type: 'boolean',
          location: 'query',
          default: false,
          description: 'defer permissions update',
        }
      }
    )
  end

  def self._update_requires_parameters
    super.merge(
      {
        async: {
          required: false,
          type: 'boolean',
          location: 'query',
          default: false,
          description: 'defer permissions update',
        }
      }
    )
  end

  def create
    if params[:async]
      @object = model_class.new(resource_attrs.merge({async_permissions_update: true}))
      @object.save!
      render_accepted
    else
      super
    end
  end

  def update
    if params[:async]
      attrs_to_update = resource_attrs.reject { |k, v|
        [:kind, :etag, :href].index k
      }.merge({async_permissions_update: true})
      @object.update!(attrs_to_update)
      @object.save!
      render_accepted
    else
      super
    end
  end

  def render_404_if_no_object
    if params[:action] == 'contents'
      if !params[:uuid]
        # OK!
        @object = nil
        true
      elsif @object
        # Project group
        true
      elsif (@object = User.where(uuid: params[:uuid]).first)
        # "Home" pseudo-project
        true
      else
        super
      end
    else
      super
    end
  end

  def contents
    @orig_select = @select
    load_searchable_objects
    list = {
      :kind => "arvados#objectList",
      :etag => "",
      :self_link => "",
      :offset => @offset,
      :limit => @limit,
      :items => @objects.as_api_response(nil)
    }
    if params[:count] != 'none'
      list[:items_available] = @items_available
    end
    if @extra_included
      list[:included] = @extra_included.as_api_response(nil, {select: @orig_select})
    end
    send_json(list)
  end

  def shared
    # The purpose of this endpoint is to return the toplevel set of
    # groups which are *not* reachable through a direct ownership
    # chain of projects starting from the current user account.  In
    # other words, groups which to which access was granted via a
    # permission link or chain of links.
    #
    # This also returns (in the "included" field) the objects that own
    # those projects (users or non-project groups).
    #
    # The intended use of this endpoint is to support clients which
    # wish to browse those projects which are visible to the user but
    # are not part of the "home" project.

    load_limit_offset_order_params
    load_filters_param

    @objects = exclude_home Group.readable_by(*@read_users), Group

    apply_where_limit_order_params

    if @include.include?("owner_uuid")
      owners = @objects.map(&:owner_uuid).to_set
      @extra_included ||= []
      [Group, User].each do |klass|
        @extra_included += klass.readable_by(*@read_users).where(uuid: owners.to_a).to_a
      end
    end

    if @include.include?("container_uuid")
      @extra_included ||= []
      container_uuids = @objects.map { |o|
        o.respond_to?(:container_uuid) ? o.container_uuid : nil
      }.compact.to_set.to_a
      @extra_included += Container.where(uuid: container_uuids).to_a
    end

    index
  end

  def self._shared_requires_parameters
    rp = self._index_requires_parameters
    rp[:include] = { type: 'string', required: false }
    rp
  end

  protected

  def load_include_param
    @include = params[:include]
    if @include.nil? || @include == ""
      @include = Set[]
    elsif @include.is_a?(String) && @include.start_with?('[')
      @include = SafeJSON.load(@include).to_set
    elsif @include.is_a?(String)
      @include = Set[@include]
    else
      return send_error("'include' parameter must be a string or array", status: 422)
    end
  end

  def load_searchable_objects
    all_objects = []
    @items_available = 0

    # Reload the orders param, this time without prefixing unqualified
    # columns ("name" => "groups.name"). Here, unqualified orders
    # apply to each table being searched, not "groups".
    load_limit_offset_order_params(fill_table_names: false)

    if params['count'] == 'none' and @offset != 0 and (params['last_object_class'].nil? or params['last_object_class'].empty?)
      # can't use offset without getting counts, so
      # fall back to count=exact behavior.
      params['count'] = 'exact'
      set_count_none = true
    end

    # Trick apply_where_limit_order_params into applying suitable
    # per-table values. *_all are the real ones we'll apply to the
    # aggregate set.
    limit_all = @limit
    offset_all = @offset
    # save the orders from the current request as determined by load_param,
    # but otherwise discard them because we're going to be getting objects
    # from many models
    request_orders = @orders.clone
    @orders = []

    request_filters = @filters

    klasses = [Group, ContainerRequest, Workflow, Collection]

    table_names = Hash[klasses.collect { |k| [k, k.table_name] }]

    disabled_methods = Rails.configuration.API.DisabledAPIs
    avail_klasses = table_names.select{|k, t| !disabled_methods[t+'.index']}
    klasses = avail_klasses.keys

    request_filters.each do |col, op, val|
      if col.index('.') && !table_names.values.include?(col.split('.', 2)[0])
        raise ArgumentError.new("Invalid attribute '#{col}' in filter")
      end
    end

    wanted_klasses = []
    request_filters.each do |col,op,val|
      if op == 'is_a'
        (val.is_a?(Array) ? val : [val]).each do |type|
          type = type.split('#')[-1]
          type[0] = type[0].capitalize
          wanted_klasses << type
        end
      end
    end

    filter_by_owner = {}
    if @object
      if params['recursive']
        filter_by_owner[:owner_uuid] = [@object.uuid] + @object.descendant_project_uuids
      else
        filter_by_owner[:owner_uuid] = @object.uuid
      end

      if params['exclude_home_project']
        raise ArgumentError.new "Cannot use 'exclude_home_project' with a parent object"
      end
    end

    # Check that any fields in @select are valid for at least one class
    if @select
      all_attributes = []
      klasses.each do |klass|
        all_attributes.concat klass.selectable_attributes
      end
      if klasses.include?(ContainerRequest) && @include.include?("container_uuid")
        all_attributes.concat Container.selectable_attributes
      end
      @select.each do |check|
        if !all_attributes.include? check
          raise ArgumentError.new "Invalid attribute '#{check}' in select"
        end
      end
    end
    any_selections = @select

    included_by_uuid = {}

    seen_last_class = false
    error_by_class = {}
    any_success = false

    klasses.each do |klass|
      # check if current klass is same as params['last_object_class']
      seen_last_class = true if((params['count'].andand.==('none')) and
                                (params['last_object_class'].nil? or
                                 params['last_object_class'].empty? or
                                 params['last_object_class'] == klass.to_s))

      # if klasses are specified, skip all other klass types
      next if wanted_klasses.any? and !wanted_klasses.include?(klass.to_s)

      # if specified, and count=none, then only look at the klass in
      # last_object_class.
      # for whatever reason, this parameter exists separately from 'wanted_klasses'
      next if params['count'] == 'none' and !seen_last_class

      # don't process rest of object types if we already have needed number of objects
      break if params['count'] == 'none' and all_objects.size >= limit_all

      # If the currently requested orders specifically match the
      # table_name for the current klass, apply that order.
      # Otherwise, order by recency.
      request_order =
        request_orders.andand.find { |r| r =~ /^#{klass.table_name}\./i || r !~ /\./ } ||
        klass.default_orders.join(", ")

      @select = select_for_klass any_selections, klass, false

      where_conds = filter_by_owner
      if klass == Collection && @select.nil?
        @select = klass.selectable_attributes - ["manifest_text", "unsigned_manifest_text"]
      elsif klass == Group
        where_conds = where_conds.merge(group_class: ["project","filter"])
      end

      # Make signed manifest_text not selectable because controller
      # currently doesn't know to sign it.
      if @select
        @select = @select - ["manifest_text"]
      end

      @filters = request_filters.map do |col, op, val|
        if !col.index('.')
          [col, op, val]
        elsif (col = col.split('.', 2))[0] == klass.table_name
          [col[1], op, val]
        else
          nil
        end
      end.compact

      @objects = klass.readable_by(*@read_users, {
          :include_trash => params[:include_trash],
          :include_old_versions => params[:include_old_versions]
        }).order(request_order).where(where_conds)

      if params['exclude_home_project']
        @objects = exclude_home @objects, klass
      end

      # Adjust the limit based on number of objects fetched so far
      klass_limit = limit_all - all_objects.count
      @limit = klass_limit

      begin
        apply_where_limit_order_params klass
      rescue ArgumentError => e
        if e.inspect =~ /Invalid attribute '.+' for operator '.+' in filter/ or
          e.inspect =~ /Invalid attribute '.+' for subproperty filter/
          error_by_class[klass.name] = e
          next
        end
        raise
      else
        any_success = true
      end

      # This actually fetches the objects
      klass_object_list = object_list(model_class: klass)

      # If count=none, :items_available will be nil, and offset is
      # required to be 0.
      klass_items_available = klass_object_list[:items_available] || 0
      @items_available += klass_items_available
      @offset = [@offset - klass_items_available, 0].max

      # Add objects to the list of objects to be returned.
      all_objects += klass_object_list[:items]

      if klass_object_list[:limit] < klass_limit
        # object_list() had to reduce @limit to comply with
        # max_index_database_read. From now on, we'll do all queries
        # with limit=0 and just accumulate items_available.
        limit_all = all_objects.count
      end

      if @include.include?("owner_uuid")
        owners = klass_object_list[:items].map {|i| i[:owner_uuid]}.to_set
        [Group, User].each do |ownerklass|
          ownerklass.readable_by(*@read_users).where(uuid: owners.to_a).each do |ow|
            included_by_uuid[ow.uuid] = ow
          end
        end
      end

      if @include.include?("container_uuid") && klass == ContainerRequest
        containers = klass_object_list[:items].collect { |cr| cr[:container_uuid] }.to_set
        Container.where(uuid: containers.to_a).each do |c|
          included_by_uuid[c.uuid] = c
        end
      end
    end

    # Only error out when every searchable object type errored out
    if !any_success && error_by_class.size > 0
      error_msg = error_by_class.collect do |klass, err|
        "#{err} on object type #{klass}"
      end.join("\n")
      raise ArgumentError.new(error_msg)
    end

    if !@include.empty?
      @extra_included = included_by_uuid.values
    end

    if set_count_none
      params['count'] = 'none'
    end

    @objects = all_objects
    @limit = limit_all
    @offset = offset_all
  end

  def exclude_home objectlist, klass
    # select records that are readable by current user AND
    #   the owner_uuid is a user (but not the current user) OR
    #   the owner_uuid is not readable by the current user
    #   the owner_uuid is a group but group_class is not a project

    read_parent_check = if current_user.is_admin
                          ""
                        else
                          "NOT EXISTS(SELECT 1 FROM #{PERMISSION_VIEW} WHERE "+
                            "user_uuid=(:user_uuid) AND target_uuid=#{klass.table_name}.owner_uuid AND perm_level >= 1) OR "
                        end

    objectlist.where("#{klass.table_name}.owner_uuid IN (SELECT users.uuid FROM users WHERE users.uuid != (:user_uuid)) OR "+
                     read_parent_check+
                     "EXISTS(SELECT 1 FROM groups as gp where gp.uuid=#{klass.table_name}.owner_uuid and gp.group_class != 'project')",
                     user_uuid: current_user.uuid)
  end
end
