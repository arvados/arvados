# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

# Mixin module for reading out query parameters from request params.
#
# Expects:
#   +params+ Hash
# Sets:
#   @where, @filters, @limit, @offset, @orders
module LoadParam

  # Default number of rows to return in a single query.
  DEFAULT_LIMIT = 100

  # Load params[:where] into @where
  def load_where_param
    if params[:where].nil? or params[:where] == ""
      @where = {}
    elsif [Hash, ActionController::Parameters].include? params[:where].class
      @where = params[:where]
    elsif params[:where].is_a? String
      begin
        @where = SafeJSON.load(params[:where])
        raise unless @where.is_a? Hash
      rescue
        raise ArgumentError.new("Could not parse \"where\" param as an object")
      end
    end
    @where = @where.with_indifferent_access
  end

  # Load params[:filters] into @filters
  def load_filters_param
    @filters ||= []
    if params[:filters].is_a? Array
      @filters += params[:filters]
    elsif params[:filters].is_a? String and !params[:filters].empty?
      begin
        f = SafeJSON.load(params[:filters])
        if not f.nil?
          raise unless f.is_a? Array
          @filters += f
        end
      rescue
        raise ArgumentError.new("Could not parse \"filters\" param as an array")
      end
    end
  end

  # Load params[:limit], params[:offset] and params[:order]
  # into @limit, @offset, @orders.
  #
  # If fill_table_names is true, @orders will be populated with fully
  # qualified columns (table_name.column_name).  Otherwise, column
  # names might be ambiguous and the caller should call
  # optimize_orders(@orders) to fix them.
  def load_limit_offset_order_params(fill_table_names: true)
    if params[:limit]
      unless params[:limit].to_s.match(/^\d+$/)
        raise ArgumentError.new("Invalid value for limit parameter")
      end
      @limit = [params[:limit].to_i,
                Rails.configuration.API.MaxItemsPerResponse].min
    else
      @limit = DEFAULT_LIMIT
    end

    if params[:offset]
      unless params[:offset].to_s.match(/^\d+$/)
        raise ArgumentError.new("Invalid value for offset parameter")
      end
      @offset = params[:offset].to_i
    else
      @offset = 0
    end

    @orders = []
    if (params[:order].is_a?(Array) && !params[:order].empty?) || !params[:order].blank?
      od = []
      (case params[:order]
       when String
         if params[:order].starts_with? '['
           od = SafeJSON.load(params[:order])
           raise unless od.is_a? Array
           od
         else
           params[:order].split(',')
         end
       when Array
         params[:order]
       else
         []
       end).each do |order|
        order = order.to_s
        attr, direction = order.strip.split " "
        direction ||= 'asc'
        # The attr can have its table unspecified if it happens to be for the current "model_class" (the first case)
        # or it can be fully specified with the database tablename (the second case) (e.g. "collections.name").
        # NB that the security check for the second case table_name will not work if the model
        # has used table_name= to use an alternate table name from the Rails standard.
        # I could not find a perfect way to handle this well, but ActiveRecord::Base.send(:descendants)
        # would be a place to start if this ever becomes necessary.
        if (attr.match(/^[a-z][_a-z0-9]+$/) &&
            model_class.columns.collect(&:name).index(attr) &&
            ['asc','desc'].index(direction.downcase))
          if fill_table_names
            @orders << "#{model_class.table_name}.#{attr} #{direction.downcase}"
          else
            @orders << "#{attr} #{direction.downcase}"
          end
        elsif attr.match(/^([a-z][_a-z0-9]+)\.([a-z][_a-z0-9]+)$/) and
            ['asc','desc'].index(direction.downcase) and
            ActiveRecord::Base.connection.tables.include?($1) and
            $1.classify.constantize.columns.collect(&:name).index($2)
          # $1 in the above checks references the first match from the regular expression, which is expected to be the database table name
          # $2 is of course the actual database column name
          @orders << "#{attr} #{direction.downcase}"
        end
      end
    end

    if fill_table_names
      @orders = optimize_orders(@orders, model_class: model_class)
    end

    @distinct = params[:distinct] && true
  end

  # If the client-specified orders don't amount to a full ordering
  # (e.g., [] or ['owner_uuid desc']), fall back on the default
  # orders to ensure repeating the same request (possibly with
  # different limit/offset) will return records in the same order.
  #
  # Clean up the resulting list of orders such that no column
  # uselessly appears twice (Postgres might not optimize this out
  # for us) and no columns uselessly appear after a unique column
  # (Postgres does not optimize this out for us; as of 9.2, "order
  # by id, modified_at desc, uuid" is slow but "order by id" is
  # fast).
  def optimize_orders(orders_given, model_class:)
    orders_given_and_default = orders_given + model_class.default_orders
    order_cols_used = {}
    optimized = []
    orders_given_and_default.each do |order|
      otablecol = order.split(' ')[0]

      next if order_cols_used[otablecol]
      order_cols_used[otablecol] = true

      optimized << order

      if otablecol.index('.')
        otable, ocol = otablecol.split('.')
      else
        otable, ocol = model_class.table_name, otablecol
      end
      if otable == model_class.table_name && model_class.unique_columns.include?(ocol)
        # we already have a full ordering; subsequent entries would be
        # superfluous
        break
      end
    end
    return optimized
  end

  def load_select_param
    case params[:select]
    when Array
      @select = params[:select]
    when String
      begin
        @select = SafeJSON.load(params[:select])
        raise unless @select.is_a? Array or @select.nil? or !@select
      rescue
        raise ArgumentError.new("Could not parse \"select\" param as an array")
      end
    end

    if @select
      # The modified_by_client_uuid field is no longer offered. For
      # the sake of compatibility with workbench2, ignore it when a
      # client asks for it explicitly (rather than returning an
      # "invalid field" error).
      @select -= ['modified_by_client_uuid']
    end

    if @select && @orders
      # Any ordering columns must be selected when doing select,
      # otherwise it is an SQL error, so filter out invaliding orderings.
      @orders.select! { |o|
        col, _ = o.split
        # match select column against order array entry
        @select.select { |s| col == "#{table_name}.#{s}" }.any?
      }
    end
  end

end
