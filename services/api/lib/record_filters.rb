# Expects:
#   @where
#   @filters
#   +model_class+
# Operates on:
#   @objects
module RecordFilters

  def record_filters filters
    cond_out = []
    param_out = []
    if filters.is_a? Array and filters.any?
      filters.each do |attr, operator, operand|
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
    end
    {:cond_out => cond_out, :param_out => param_out}
  end

end
