# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

# Mixin module providing a method to convert filters into a list of SQL
# fragments suitable to be fed to ActiveRecord #where.
#
# Expects:
#   model_class
# Operates on:
#   @objects

require 'safe_json'

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
        if operand.is_a? Array
          raise ArgumentError.new("Full text search not supported for array operands")
        end

        # Skip the generic per-column operator loop below
        attrs = []
        # Use to_tsquery since plainto_tsquery does not support prefix
        # search. And, split operand and join the words with ' & '
        cond_out << model_class.full_text_tsvector+" @@ to_tsquery(?)"
        param_out << operand.split.join(' & ')
      end
      attrs.each do |attr|
        subproperty = attr.split(".", 2)

        col = model_class.columns.select { |c| c.name == subproperty[0] }.first

        if subproperty.length == 2
          if col.nil? or col.type != :jsonb
            raise ArgumentError.new("Invalid attribute '#{subproperty[0]}' for subproperty filter")
          end

          if subproperty[1][0] == "<" and subproperty[1][-1] == ">"
            subproperty[1] = subproperty[1][1..-2]
          end

        # jsonb search
          case operator.downcase
          when '=', '!='
            not_in = if operator.downcase == "!=" then "NOT " else "" end
            cond_out << "#{not_in}(#{ar_table_name}.#{subproperty[0]} @> ?::jsonb)"
            param_out << SafeJSON.dump({subproperty[1] => operand})
          when 'in'
            if operand.is_a? Array
              operand.each do |opr|
                cond_out << "#{ar_table_name}.#{subproperty[0]} @> ?::jsonb"
                param_out << SafeJSON.dump({subproperty[1] => opr})
              end
            else
              raise ArgumentError.new("Invalid operand type '#{operand.class}' "\
                                      "for '#{operator}' operator in filters")
            end
          when '<', '<=', '>', '>='
            cond_out << "#{ar_table_name}.#{subproperty[0]}->? #{operator} ?::jsonb"
            param_out << subproperty[1]
            param_out << SafeJSON.dump(operand)
          when 'like', 'ilike'
            cond_out << "#{ar_table_name}.#{subproperty[0]}->>? #{operator} ?"
            param_out << subproperty[1]
            param_out << operand
          when 'not in'
            if operand.is_a? Array
              cond_out << "#{ar_table_name}.#{subproperty[0]}->>? NOT IN (?) OR #{ar_table_name}.#{subproperty[0]}->>? IS NULL"
              param_out << subproperty[1]
              param_out << operand
              param_out << subproperty[1]
            else
              raise ArgumentError.new("Invalid operand type '#{operand.class}' "\
                                      "for '#{operator}' operator in filters")
            end
          when 'exists'
          if operand == true
            cond_out << "jsonb_exists(#{ar_table_name}.#{subproperty[0]}, ?)"
          elsif operand == false
            cond_out << "(NOT jsonb_exists(#{ar_table_name}.#{subproperty[0]}, ?)) OR #{ar_table_name}.#{subproperty[0]} is NULL"
          else
            raise ArgumentError.new("Invalid operand '#{operand}' for '#{operator}' must be true or false")
          end
          param_out << subproperty[1]
          else
            raise ArgumentError.new("Invalid operator for subproperty search '#{operator}'")
          end
        elsif operator.downcase == "exists"
          if col.type != :jsonb
            raise ArgumentError.new("Invalid attribute '#{subproperty[0]}' for operator '#{operator}' in filter")
          end

          cond_out << "jsonb_exists(#{ar_table_name}.#{subproperty[0]}, ?)"
          param_out << operand
        else
          if !model_class.searchable_columns(operator).index subproperty[0]
            raise ArgumentError.new("Invalid attribute '#{subproperty[0]}' in filter")
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
              if operator == '<>'
                # explicitly allow NULL
                cond_out << "#{ar_table_name}.#{attr} #{operator} ? OR #{ar_table_name}.#{attr} IS NULL"
              else
                cond_out << "#{ar_table_name}.#{attr} #{operator} ?"
              end
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
            elsif (attr_type == :integer)
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
          else
            raise ArgumentError.new("Invalid operator '#{operator}'")
          end
        end
      end
      conds_out << cond_out.join(' OR ') if cond_out.any?
    end

    {:cond_out => conds_out, :param_out => param_out}
  end

end
