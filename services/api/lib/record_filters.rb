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

      operator = operator.downcase
      cond_out = []

      if attrs_in == 'any' && (operator == 'ilike' || operator == 'like') && (operand.is_a? String) && operand.match('^[%].*[%]$')
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
          case operator
          when '=', '!='
            not_in = if operator == "!=" then "NOT " else "" end
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
              cond_out << "jsonb_exists_inline_op(#{attr_table_name}.#{attr}, ?)"
            elsif operand == false
              cond_out << "(NOT jsonb_exists_inline_op(#{attr_table_name}.#{attr}, ?)) OR #{attr_table_name}.#{attr} is NULL"
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
        elsif operator == "exists"
          if col.nil? or col.type != :jsonb
            raise ArgumentError.new("Invalid attribute '#{attr}' for operator '#{operator}' in filter")
          end

          cond_out << "jsonb_exists_inline_op(#{attr_table_name}.#{attr}, ?)"
          param_out << operand
        elsif expr = /^ *\( *(\w+) *(<=?|>=?|=) *(\w+) *\) *$/.match(attr)
          if operator != '=' || ![true,"true"].index(operand)
            raise ArgumentError.new("Invalid expression filter '#{attr}': subsequent elements must be [\"=\", true]")
          end
          operator = expr[2]
          attr1, attr2 = expr[1], expr[3]
          allowed = attr_model_class.searchable_columns(operator)
          [attr1, attr2].each do |tok|
            if !allowed.index(tok)
              raise ArgumentError.new("Invalid attribute in expression: '#{tok}'")
            end
            col = attr_model_class.columns.select { |c| c.name == tok }.first
            if col.type != :integer
              raise ArgumentError.new("Non-numeric attribute in expression: '#{tok}'")
            end
          end
          cond_out << "#{attr1} #{operator} #{attr2}"
        else
          if !attr_model_class.searchable_columns(operator).index(attr) &&
             !(col.andand.type == :jsonb && ['contains', '=', '<>', '!='].index(operator))
            raise ArgumentError.new("Invalid attribute '#{attr}' in filter")
          end
          attr_type = attr_model_class.attribute_column(attr).type

          case operator
          when '=', '<', '<=', '>', '>=', '!=', 'like', 'ilike'
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
                  raise ArgumentError.new("Invalid operand '#{operand}' for " \
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
              if !operand.is_a?(Integer) || operand.bit_length > 64
                raise ArgumentError.new("Invalid operand '#{operand}' "\
                                        "for integer attribute '#{attr}'")
              end
              cond_out << "#{attr_table_name}.#{attr} #{operator} ?"
              param_out << operand
            else
              raise ArgumentError.new("Invalid operand type '#{operand.class}' "\
                                      "for '#{operator}' operator in filters")
            end
          when 'in', 'not in'
            if !operand.is_a? Array
              raise ArgumentError.new("Invalid operand type '#{operand.class}' "\
                                      "for '#{operator}' operator in filters")
            end
            if attr_type == :integer
              operand.each do |el|
                if !el.is_a?(Integer) || el.bit_length > 64
                  raise ArgumentError.new("Invalid element '#{el}' in array "\
                                          "for integer attribute '#{attr}'")
                end
              end
            end
            cond_out << "#{attr_table_name}.#{attr} #{operator} (?)"
            param_out << operand
            if operator == 'not in' and not operand.include?(nil)
              # explicitly allow NULL
              cond_out[-1] = "(#{cond_out[-1]} OR #{attr_table_name}.#{attr} IS NULL)"
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
          when 'contains'
            if col.andand.type != :jsonb
              raise ArgumentError.new("Invalid attribute '#{attr}' for '#{operator}' operator")
            end
            if operand == []
              raise ArgumentError.new("Invalid operand '#{operand.inspect}' for '#{operator}' operator")
            end
            operand = [operand] unless operand.is_a? Array
            operand.each do |op|
              if !op.is_a?(String)
                raise ArgumentError.new("Invalid element #{operand.inspect} in operand for #{operator.inspect} operator (operand must be a string or array of strings)")
              end
            end
            # We use jsonb_exists_all_inline_op(a,b) instead of "a ?&
            # b" because the pg gem thinks "?" is a bind var.
            #
            # See note in migration
            # 20230815160000_jsonb_exists_functions about _inline_op
            # functions.
            #
            # We use string interpolation instead of param_out
            # because the pg gem flattens param_out / doesn't support
            # passing arrays as bind vars.
            q = operand.map { |s| ActiveRecord::Base.connection.quote(s) }.join(',')
            cond_out << "jsonb_exists_all_inline_op(#{attr_table_name}.#{attr}, array[#{q}])"
          else
            raise ArgumentError.new("Invalid operator '#{operator}'")
          end
        end
      end
      conds_out << cond_out.join(' OR ') if cond_out.any?
    end

    {:cond_out => conds_out, :param_out => param_out, :joins => joins}
  end

  def apply_filters query, filters
    ft = record_filters filters, self
    if not ft[:cond_out].any?
      return query
    end
    ft[:joins].each do |t|
      query = query.joins(t)
    end
    query.where('(' + ft[:cond_out].join(') AND (') + ')',
                          *ft[:param_out])
  end

  def attribute_column attr
    self.columns.select { |col| col.name == attr.to_s }.first
  end
end
