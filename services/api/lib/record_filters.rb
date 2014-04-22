# Expects:
#   @where
#   @filters
#   +model_class+
# Operates on:
#   @objects
module RecordFilters

  def apply_where_limit_order_params
    if @filters.is_a? Array and @filters.any?
      cond_out = []
      param_out = []
      @filters.each do |attr, operator, operand|
        if !model_class.searchable_columns(operator).index attr.to_s
          raise ArgumentError.new("Invalid attribute '#{attr}' in condition")
        end
        case operator.downcase
        when '=', '<', '<=', '>', '>=', 'like'
          if operand.is_a? String
            cond_out << "#{table_name}.#{attr} #{operator} ?"
            if (# any operator that operates on value rather than
                # representation:
                operator.match(/[<=>]/) and
                model_class.attribute_column(attr).type == :datetime)
              operand = Time.parse operand
            end
            param_out << operand
          end
        when 'in'
          if operand.is_a? Array
            cond_out << "#{table_name}.#{attr} IN (?)"
            param_out << operand
          end
        when 'is_a'
          operand = [operand] unless operand.is_a? Array
          cond = []
          operand.each do |op|
              cl = ArvadosModel::kind_class op
              if cl
                cond << "#{table_name}.#{attr} like ?"
                param_out << cl.uuid_like_pattern
              else
                cond << "1=0"
              end
          end
          cond_out << cond.join(' OR ')
        end
      end
      if cond_out.any?
        @objects = @objects.where(cond_out.join(' AND '), *param_out)
      end
    end
    if @where.is_a? Hash and @where.any?
      conditions = ['1=1']
      @where.each do |attr,value|
        if attr.to_s == 'any'
          if value.is_a?(Array) and
              value.length == 2 and
              value[0] == 'contains' then
            ilikes = []
            model_class.searchable_columns('ilike').each do |column|
              ilikes << "#{table_name}.#{column} ilike ?"
              conditions << "%#{value[1]}%"
            end
            if ilikes.any?
              conditions[0] << ' and (' + ilikes.join(' or ') + ')'
            end
          end
        elsif attr.to_s.match(/^[a-z][_a-z0-9]+$/) and
            model_class.columns.collect(&:name).index(attr.to_s)
          if value.nil?
            conditions[0] << " and #{table_name}.#{attr} is ?"
            conditions << nil
          elsif value.is_a? Array
            if value[0] == 'contains' and value.length == 2
              conditions[0] << " and #{table_name}.#{attr} like ?"
              conditions << "%#{value[1]}%"
            else
              conditions[0] << " and #{table_name}.#{attr} in (?)"
              conditions << value
            end
          elsif value.is_a? String or value.is_a? Fixnum or value == true or value == false
            conditions[0] << " and #{table_name}.#{attr}=?"
            conditions << value
          elsif value.is_a? Hash
            # Not quite the same thing as "equal?" but better than nothing?
            value.each do |k,v|
              if v.is_a? String
                conditions[0] << " and #{table_name}.#{attr} ilike ?"
                conditions << "%#{k}%#{v}%"
              end
            end
          end
        end
      end
      if conditions.length > 1
        conditions[0].sub!(/^1=1 and /, '')
        @objects = @objects.
          where(*conditions)
      end
    end

    if params[:limit]
      begin
        @limit = params[:limit].to_i
      rescue
        raise ArgumentError.new("Invalid value for limit parameter")
      end
    else
      @limit = 100
    end
    @objects = @objects.limit(@limit)

    orders = []

    if params[:offset]
      begin
        @objects = @objects.offset(params[:offset].to_i)
        @offset = params[:offset].to_i
      rescue
        raise ArgumentError.new("Invalid value for limit parameter")
      end
    else
      @offset = 0
    end

    orders = []
    if params[:order]
      params[:order].split(',').each do |order|
        attr, direction = order.strip.split " "
        direction ||= 'asc'
        if attr.match /^[a-z][_a-z0-9]+$/ and
            model_class.columns.collect(&:name).index(attr) and
            ['asc','desc'].index direction.downcase
          orders << "#{table_name}.#{attr} #{direction.downcase}"
        end
      end
    end
    if orders.empty?
      orders << "#{table_name}.modified_at desc"
    end
    @objects = @objects.order(orders.join ", ")
  end

end
