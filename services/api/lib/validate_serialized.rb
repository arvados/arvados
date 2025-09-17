# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

# HashAttrValidator is invoked via "validates :attr_name,
# array_of_strings: {allow_empty_strings: true}".
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

# HashAttrValidator is invoked via "validates :attr_name, hash_attr:
# true".  We don't call it just HashValidator because that conflicts
# with the hash_validator gem.
class HashAttrValidator < ActiveModel::EachValidator
  def validate_each(record, attr, val)
    if !val.is_a?(Hash)
      record.errors.add attr, "must be a hash"
      return
    end
  end
end
