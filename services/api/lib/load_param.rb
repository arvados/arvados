# Mixin module for reading out query parameters from request params.
#
# Expects:
#   +params+ Hash
# Sets:
#   @where, @filters, @limit, @offset, @orders
module LoadParam

  # Default limit on number of rows to return in a single query.
  DEFAULT_LIMIT = 100

  # Load params[:where] into @where
  def load_where_param
    if params[:where].nil? or params[:where] == ""
      @where = {}
    elsif params[:where].is_a? Hash
      @where = params[:where]
    elsif params[:where].is_a? String
      begin
        @where = Oj.load(params[:where])
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
        f = Oj.load params[:filters]
        raise unless f.is_a? Array
        @filters += f
      rescue
        raise ArgumentError.new("Could not parse \"filters\" param as an array")
      end
    end
  end

  def default_orders
    ["#{table_name}.modified_at desc"]
  end

  # Load params[:limit], params[:offset] and params[:order]
  # into @limit, @offset, @orders
  def load_limit_offset_order_params
    if params[:limit]
      unless params[:limit].to_s.match(/^\d+$/)
        raise ArgumentError.new("Invalid value for limit parameter")
      end
      @limit = params[:limit].to_i
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
    if params[:order]
      params[:order].split(',').each do |order|
        attr, direction = order.strip.split " "
        direction ||= 'asc'
        if attr.match /^[a-z][_a-z0-9]+$/ and
            model_class.columns.collect(&:name).index(attr) and
            ['asc','desc'].index direction.downcase
          @orders << "#{table_name}.#{attr} #{direction.downcase}"
        end
      end
    end
    if @orders.empty?
      @orders << default_orders
    end

    case params[:select]
    when Array
      @select = params[:select]
    when String
      begin
        @select = Oj.load params[:select]
        raise unless @select.is_a? Array
      rescue
        raise ArgumentError.new("Could not parse \"select\" param as an array")
      end
    end

    if params[:distinct].is_a? String
      @distinct = params[:distinct]
      @orders.select! { |o| @select.include? o } if @select
    end
  end


end
