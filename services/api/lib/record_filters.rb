# Expects:
#   @where
#   @filters
#   +model_class+
# Operates on:
#   @objects
module RecordFilters

  def record_filters filters, ar_table_name
    cond_out = []
    param_out = []

    filters.each do |filter|
      attr, operator, operand = filter
      if !filter.is_a? Array
        raise ArgumentError.new("Invalid element in filters array: #{filter.inspect} is not an array")
      elsif !operator.is_a? String
        raise ArgumentError.new("Invalid operator '#{operator}' (#{operator.class}) in filter")
      elsif !model_class.searchable_columns(operator).index attr.to_s
        raise ArgumentError.new("Invalid attribute '#{attr}' in filter")
      end
      case operator.downcase
      when '=', '<', '<=', '>', '>=', 'like'
        if operand.is_a? String
          cond_out << "#{ar_table_name}.#{attr} #{operator} ?"
          if (# any operator that operates on value rather than
              # representation:
              operator.match(/[<=>]/) and
              model_class.attribute_column(attr).type == :datetime)
            operand = Time.parse operand
          end
          param_out << operand
        elsif operand.nil? and operator == '='
          cond_out << "#{ar_table_name}.#{attr} is null"
        else
          raise ArgumentError.new("Invalid operand type '#{operand.class}' "\
                                  "for '#{operator}' operator in filters")
        end
      when 'in'
        if operand.is_a? Array
          cond_out << "#{ar_table_name}.#{attr} IN (?)"
          param_out << operand
        else
          raise ArgumentError.new("Invalid operand type '#{operand.class}' "\
                                  "for '#{operator}' operator in filters")
        end
      when 'is_a'
        operand = [operand] unless operand.is_a? Array
        cond = []
        operand.each do |op|
          cl = ArvadosModel::kind_class op
          if cl
            cond << "#{ar_table_name}.#{attr} like ?"
            param_out << cl.uuid_like_pattern
          else
            cond << "1=0"
          end
        end
        cond_out << cond.join(' OR ')
      end
    end

    {:cond_out => cond_out, :param_out => param_out}
  end

end
