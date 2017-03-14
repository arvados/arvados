class Arvados::V1::GroupsController < ApplicationController

  def self._contents_requires_parameters
    params = _index_requires_parameters.
      merge({
              uuid: {
                type: 'string', required: false, default: nil
              },
            })
    params.delete(:select)
    params
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
    load_searchable_objects
    send_json({
      :kind => "arvados#objectList",
      :etag => "",
      :self_link => "",
      :offset => @offset,
      :limit => @limit,
      :items_available => @items_available,
      :items => @objects.as_api_response(nil)
    })
  end

  protected

  def load_searchable_objects
    all_objects = []
    @items_available = 0

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

    klasses = [Group,
     Job, PipelineInstance, PipelineTemplate, ContainerRequest, Workflow,
     Collection,
     Human, Specimen, Trait]

    table_names = Hash[klasses.collect { |k| [k, k.table_name] }]

    disabled_methods = Rails.configuration.disable_api_methods
    avail_klasses = table_names.select{|k, t| !disabled_methods.include?(t+'.index')}
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

    seen_last_class = false
    klasses.each do |klass|
      @offset = 0 if seen_last_class  # reset offset for the new next type being processed

      # if current klass is same as params['last_object_class'], mark that fact
      seen_last_class = true if((params['count'].andand.==('none')) and
                                (params['last_object_class'].nil? or
                                 params['last_object_class'].empty? or
                                 params['last_object_class'] == klass.to_s))

      # if klasses are specified, skip all other klass types
      next if wanted_klasses.any? and !wanted_klasses.include?(klass.to_s)

      # don't reprocess klass types that were already seen
      next if params['count'] == 'none' and !seen_last_class

      # don't process rest of object types if we already have needed number of objects
      break if params['count'] == 'none' and all_objects.size >= limit_all

      # If the currently requested orders specifically match the
      # table_name for the current klass, apply that order.
      # Otherwise, order by recency.
      request_order =
        request_orders.andand.find { |r| r =~ /^#{klass.table_name}\./i } ||
        klass.default_orders.join(", ")

      @select = nil
      where_conds = {}
      where_conds[:owner_uuid] = @object.uuid if @object
      if klass == Collection
        @select = klass.selectable_attributes - ["manifest_text"]
      elsif klass == Group
        where_conds[:group_class] = "project"
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

      @objects = klass.readable_by(*@read_users).
        order(request_order).where(where_conds)
      @limit = limit_all - all_objects.count
      apply_where_limit_order_params klass
      klass_object_list = object_list
      klass_items_available = klass_object_list[:items_available] || 0
      @items_available += klass_items_available
      @offset = [@offset - klass_items_available, 0].max
      all_objects += klass_object_list[:items]
    end

    @objects = all_objects
    @limit = limit_all
    @offset = offset_all
  end

end
