# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

class ArrayOfStringsValidator < ActiveModel::EachValidator
  def validate_each(record, attr, val)
    if !val.is_a?(Array)
      record.errors.add attr, "must be an array of non-empty strings"
      return
    end
    val.each do |ent|
      if !ent.is_a?(String) || ent == ""
        record.errors.add attr, "must be an array of non-empty strings"
        return
      end
    end
  end
end

class HashValidator < ActiveModel::EachValidator
  def validate_each(record, attr, val)
    if !val.is_a?(Hash)
      record.errors.add attr, "must be a hash"
      return
    end
  end
end
