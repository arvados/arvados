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
  # Hash with the following keys:
  # :cond_out  array of SQL fragments for each filter expression
  # :param_out array of values for parameter substitution in cond_out
  # :joins     array of joins: either [] or ["JOIN containers ON ..."]
  def record_filters filters, model_class
    conds_out = []
    param_out = []
    joins = []

    model_table_name = model_class.table_name
    filters.each do |filter|
      attrs_in, operator, operand = filter
      if operator == '@@'
        raise ArgumentError.new("Full text search operator is no longer supported")
      end
      if attrs_in == 'any'
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

      if attrs_in == 'any' && (operator.casecmp('ilike').zero? || operator.casecmp('like').zero?) && (operand.is_a? String) && operand.match('^[%].*[%]$')
        # Trigram index search
        cond_out << model_class.full_text_trgm + " #{operator} ?"
        param_out << operand
        # Skip the generic per-column operator loop below
        attrs = []
      end

      attrs.each do |attr|
        subproperty = attr.split(".", 2)

        if subproperty.length == 2 && subproperty[0] == 'container' && model_table_name == "container_requests"
          # attr is "tablename.colname" -- e.g., ["container.state", "=", "Complete"]
          joins = ["JOIN containers ON container_requests.container_uuid = containers.uuid"]
          attr_model_class = Container
          attr_table_name = "containers"
          subproperty = subproperty[1].split(".", 2)
        else
          attr_model_class = model_class
          attr_table_name = model_table_name
        end

        attr = subproperty[0]
        proppath = subproperty[1]
        col = attr_model_class.columns.select { |c| c.name == attr }.first

        if proppath
          if col.nil? or col.type != :jsonb
            raise ArgumentError.new("Invalid attribute '#{attr}' for subproperty filter")
          end

          if proppath[0] == "<" and proppath[-1] == ">"
            proppath = proppath[1..-2]
          end

          # jsonb search
          case operator.downcase
          when '=', '!='
            not_in = if operator.downcase == "!=" then "NOT " else "" end
            cond_out << "#{not_in}(#{attr_table_name}.#{attr} @> ?::jsonb)"
            param_out << SafeJSON.dump({proppath => operand})
          when 'in'
            if operand.is_a? Array
              operand.each do |opr|
                cond_out << "#{attr_table_name}.#{attr} @> ?::jsonb"
                param_out << SafeJSON.dump({proppath => opr})
              end
            else
              raise ArgumentError.new("Invalid operand type '#{operand.class}' "\
                                      "for '#{operator}' operator in filters")
            end
          when '<', '<=', '>', '>='
            cond_out << "#{attr_table_name}.#{attr}->? #{operator} ?::jsonb"
            param_out << proppath
            param_out << SafeJSON.dump(operand)
          when 'like', 'ilike'
            cond_out << "#{attr_table_name}.#{attr}->>? #{operator} ?"
            param_out << proppath
            param_out << operand
          when 'not in'
            if operand.is_a? Array
              cond_out << "#{attr_table_name}.#{attr}->>? NOT IN (?) OR #{attr_table_name}.#{attr}->>? IS NULL"
              param_out << proppath
              param_out << operand
              param_out << proppath
            else
              raise ArgumentError.new("Invalid operand type '#{operand.class}' "\
                                      "for '#{operator}' operator in filters")
            end
          when 'exists'
            if operand == true
              cond_out << "jsonb_exists(#{attr_table_name}.#{attr}, ?)"
            elsif operand == false
              cond_out << "(NOT jsonb_exists(#{attr_table_name}.#{attr}, ?)) OR #{attr_table_name}.#{attr} is NULL"
            else
              raise ArgumentError.new("Invalid operand '#{operand}' for '#{operator}' must be true or false")
            end
            param_out << proppath
          when 'contains'
            cond_out << "#{attr_table_name}.#{attr} @> ?::jsonb OR #{attr_table_name}.#{attr} @> ?::jsonb"
            param_out << SafeJSON.dump({proppath => operand})
            param_out << SafeJSON.dump({proppath => [operand]})
          else
            raise ArgumentError.new("Invalid operator for subproperty search '#{operator}'")
          end
        elsif operator.downcase == "exists"
          if col.type != :jsonb
            raise ArgumentError.new("Invalid attribute '#{attr}' for operator '#{operator}' in filter")
          end

          cond_out << "jsonb_exists(#{attr_table_name}.#{attr}, ?)"
          param_out << operand
        else
          if !attr_model_class.searchable_columns(operator).index attr
            raise ArgumentError.new("Invalid attribute '#{attr}' in filter")
          end

          case operator.downcase
          when '=', '<', '<=', '>', '>=', '!=', 'like', 'ilike'
            attr_type = attr_model_class.attribute_column(attr).type
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
                cond_out << "#{attr_table_name}.#{attr} #{operator} ? OR #{attr_table_name}.#{attr} IS NULL"
              else
                cond_out << "#{attr_table_name}.#{attr} #{operator} ?"
              end
              if (# any operator that operates on value rather than
                # representation:
                operator.match(/[<=>]/) and (attr_type == :datetime))
                operand = Time.parse operand
              end
              param_out << operand
            elsif operand.nil? and operator == '='
              cond_out << "#{attr_table_name}.#{attr} is null"
            elsif operand.nil? and operator == '<>'
              cond_out << "#{attr_table_name}.#{attr} is not null"
            elsif (attr_type == :boolean) and ['=', '<>'].include?(operator) and
                 [true, false].include?(operand)
              cond_out << "#{attr_table_name}.#{attr} #{operator} ?"
              param_out << operand
            elsif (attr_type == :integer)
              cond_out << "#{attr_table_name}.#{attr} #{operator} ?"
              param_out << operand
            else
              raise ArgumentError.new("Invalid operand type '#{operand.class}' "\
                                      "for '#{operator}' operator in filters")
            end
          when 'in', 'not in'
            if operand.is_a? Array
              cond_out << "#{attr_table_name}.#{attr} #{operator} (?)"
              param_out << operand
              if operator == 'not in' and not operand.include?(nil)
                # explicitly allow NULL
                cond_out[-1] = "(#{cond_out[-1]} OR #{attr_table_name}.#{attr} IS NULL)"
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
                if attr == 'uuid'
                  if attr_model_class.uuid_prefix == cl.uuid_prefix
                    cond << "1=1"
                  else
                    cond << "1=0"
                  end
                else
                  # Use a substring query to support remote uuids
                  cond << "substring(#{attr_table_name}.#{attr}, 7, 5) = ?"
                  param_out << cl.uuid_prefix
                end
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

    {:cond_out => conds_out, :param_out => param_out, :joins => joins}
  end

end
