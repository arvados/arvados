# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

class JsonbType
  # Emulate pre-rails5.0 behavior by having a interpreting NULL/nil as
  # some other default value.
  class WithDefault < ActiveModel::Type::Value
    include ActiveModel::Type::Helpers::Mutable

    def default_value
      nil
    end

    def deserialize(value)
      if value.nil?
        self.default_value
      elsif value.is_a?(::String)
        SafeJSON.load(value) rescue self.default_value
      else
        value
      end
    end

    def serialize(value)
      if value.nil?
        self.default_value
      else
        SafeJSON.dump(value)
      end
    end
  end

  class Hash < JsonbType::WithDefault
    def default_value
      {}
    end
  end

  class Array < JsonbType::WithDefault
    def default_value
      []
    end
  end
end