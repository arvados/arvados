# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

# ArrayOfStringsValidator is invoked via:
#     validates :attr_name, array_of_strings: {allow_empty_strings: true}
class ArrayOfStringsValidator < ActiveModel::EachValidator
  def validate_each(record, attr, val)
    errtxt = "must be an array of #{"non-empty " unless options[:allow_empty_strings]}strings"
    if !val.is_a?(Array)
      record.errors.add attr, errtxt
      return
    end
    val.each do |ent|
      if !ent.is_a?(String) || (ent == "" && !options[:allow_empty_strings])
        record.errors.add attr, errtxt
        return
      end
    end
  end
end

# HashAttrValidator is invoked via:
#     validates :attr_name, hash_attr: true
# We don't call it just HashValidator because that conflicts with the
# hash_validator gem.
class HashAttrValidator < ActiveModel::EachValidator
  def validate_each(record, attr, val)
    if !val.is_a?(Hash)
      record.errors.add attr, "must be a hash"
      return
    end
    # Note this should be a no-op since JSON is the only way we accept
    # serialized attributes.  But we still explicitly check that keys
    # are strings, etc., before running code that assumes so.
    ensure_json_representable(record, attr, val)
  end

  def ensure_json_representable(record, attr, v)
    case v
    when String, Integer, Float, BigDecimal, Boolean, NilClass
      return true
    when Array
      v.each do |v|
        if !ensure_json_representable(record, attr, v)
          return false
        end
      end
    when Hash
      v.each do |k, v|
        if !k.is_a?(String)
          record.errors.add attr, "contains non-string hash key #{k.inspect}"
          return false
        end
        if !ensure_json_representable(record, attr, v)
          return false
        end
      end
    else
      record.errors.add attr, "contains value #{v.inspect} with unexpected class #{v.class}"
      return false
    end
  end
end
