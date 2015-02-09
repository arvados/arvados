# Mixin module providing a method to convert filters into a list of SQL
# fragments suitable to be fed to ActiveRecord #where.
#
# Expects:
#   model_class
# Operates on:
#   @objects
module RecordFilters

  # Input:
  # +filters+        array of conditions, each being [column, operator, operand]
  # +model_class+    subclass of ActiveRecord being filtered
  #
  # Output:
  # Hash with two keys:
  # :cond_out  array of SQL fragments for each filter expression
  # :param_out  array of values for parameter substitution in cond_out
  def record_filters filters, model_class
    conds_out = []
    param_out = []

    ar_table_name = model_class.table_name
    filters.each do |filter|
      attrs_in, operator, operand = filter
      if attrs_in == 'any' && operator != '@@'
        attrs = model_class.searchable_columns(operator)
      elsif attrs_in.is_a? Array
        attrs = attrs_in
      else
        attrs = [attrs_in]
      end
      if !filter.is_a? Array
        raise ArgumentError.new("Invalid element in filters array: #{filter.inspect} is not an array")
      elsif !operator.is_a? String
        raise ArgumentError.new("Invalid operator '#{operator}' (#{operator.class}) in filter")
      end

      cond_out = []

      if operator == '@@'
        # Full-text search
        if attrs_in != 'any'
          raise ArgumentError.new("Full text search on individual columns is not supported")
        end
        attrs = [] #  skip the generic per-column operator loop below
        # Use to_tsquery since plainto_tsquery does not support prefix search.
        # Instead split operand, add ':*' to each word and join the words with ' & '
        cond_out << model_class.full_text_tsvector+" @@ to_tsquery(?)"
        param_out << operand.split.each {|s| s.concat(':*')}.join(' & ')
      end
      attrs.each do |attr|
        if !model_class.searchable_columns(operator).index attr.to_s
          raise ArgumentError.new("Invalid attribute '#{attr}' in filter")
        end
        case operator.downcase
        when '=', '<', '<=', '>', '>=', '!=', 'like', 'ilike'
          attr_type = model_class.attribute_column(attr).type
          operator = '<>' if operator == '!='
          if operand.is_a? String
            if attr_type == :boolean
              if not ['=', '<>'].include?(operator)
                raise ArgumentError.new("Invalid operator '#{operator}' for " \
                                        "boolean attribute '#{attr}'")
              end
              case operand.downcase
              when '1', 't', 'true', 'y', 'yes'
                operand = true
              when '0', 'f', 'false', 'n', 'no'
                operand = false
              else
                raise ArgumentError("Invalid operand '#{operand}' for " \
                                    "boolean attribute '#{attr}'")
              end
            end
            cond_out << "#{ar_table_name}.#{attr} #{operator} ?"
            if (# any operator that operates on value rather than
                # representation:
                operator.match(/[<=>]/) and (attr_type == :datetime))
              operand = Time.parse operand
            end
            param_out << operand
          elsif operand.nil? and operator == '='
            cond_out << "#{ar_table_name}.#{attr} is null"
          elsif operand.nil? and operator == '<>'
            cond_out << "#{ar_table_name}.#{attr} is not null"
          elsif (attr_type == :boolean) and ['=', '<>'].include?(operator) and
              [true, false].include?(operand)
            cond_out << "#{ar_table_name}.#{attr} #{operator} ?"
            param_out << operand
          else
            raise ArgumentError.new("Invalid operand type '#{operand.class}' "\
                                    "for '#{operator}' operator in filters")
          end
        when 'in', 'not in'
          if operand.is_a? Array
            cond_out << "#{ar_table_name}.#{attr} #{operator} (?)"
            param_out << operand
            if operator == 'not in' and not operand.include?(nil)
              # explicitly allow NULL
              cond_out[-1] = "(#{cond_out[-1]} OR #{ar_table_name}.#{attr} IS NULL)"
            end
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
      conds_out << cond_out.join(' OR ') if cond_out.any?
    end

    {:cond_out => conds_out, :param_out => param_out}
  end

end
