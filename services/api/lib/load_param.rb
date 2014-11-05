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
        if not f.nil?
          raise unless f.is_a? Array
          @filters += f
        end
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
    if (params[:order].is_a?(Array) && !params[:order].empty?) || !params[:order].blank?
      od = []
      (case params[:order]
       when String
         if params[:order].starts_with? '['
           od = Oj.load(params[:order])
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
        # has used set_table_name to use an alternate table name from the Rails standard.
        # I could not find a perfect way to handle this well, but ActiveRecord::Base.send(:descendants)
        # would be a place to start if this ever becomes necessary.
        if attr.match /^[a-z][_a-z0-9]+$/ and
            model_class.columns.collect(&:name).index(attr) and
            ['asc','desc'].index direction.downcase
          @orders << "#{table_name}.#{attr} #{direction.downcase}"
        elsif attr.match /^([a-z][_a-z0-9]+)\.([a-z][_a-z0-9]+)$/ and
            ['asc','desc'].index(direction.downcase) and
            ActiveRecord::Base.connection.tables.include?($1) and
            $1.classify.constantize.columns.collect(&:name).index($2)
          # $1 in the above checks references the first match from the regular expression, which is expected to be the database table name
          # $2 is of course the actual database column name
          @orders << "#{attr} #{direction.downcase}"
        end
      end
    end

    if @orders.empty?
      @orders = default_orders
    end

    case params[:select]
    when Array
      @select = params[:select]
    when String
      begin
        @select = Oj.load params[:select]
        raise unless @select.is_a? Array or @select.nil?
      rescue
        raise ArgumentError.new("Could not parse \"select\" param as an array")
      end
    end

    if @select
      # Any ordering columns must be selected when doing select,
      # otherwise it is an SQL error, so filter out invaliding orderings.
      @orders.select! { |o|
        # match select column against order array entry
        @select.select { |s| /^#{table_name}.#{s}( (asc|desc))?$/.match o }.any?
      }
    end

    @distinct = true if (params[:distinct] == true || params[:distinct] == "true")
    @distinct = false if (params[:distinct] == false || params[:distinct] == "false")
  end

end
