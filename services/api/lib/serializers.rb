# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

require 'safe_json'

class Serializer
  class TypeMismatch < ArgumentError
  end

  def self.dump(val)
    if !val.is_a?(object_class)
      raise TypeMismatch.new("cannot serialize #{val.class} as #{object_class}")
    end
    SafeJSON.dump(val)
  end

  def self.legacy_load(s)
    val = Psych.safe_load(s, permitted_classes: [Time])
    if val.is_a? String
      # If apiserver was downgraded to a YAML-only version after
      # storing JSON in the database, the old code would have loaded
      # the JSON document as a plain string, and then YAML-encoded
      # it when saving it back to the database. It's too late now to
      # make the old code behave better, but at least we can
      # gracefully handle the mess it leaves in the database by
      # double-decoding on the way out.
      return SafeJSON.load(val)
    else
      return val
    end
  end

  def self.load(s)
    if s.is_a?(object_class)
      # Rails already deserialized for us
      s
    elsif s.nil?
      object_class.new()
    elsif s[0] == first_json_char
      SafeJSON.load(s)
    elsif s[0..2] == "---"
      legacy_load(s)
    else
      raise "invalid serialized data #{s[0..5].inspect}"
    end
  end
end

class HashSerializer < Serializer
  def self.first_json_char
    "{"
  end

  def self.object_class
    ::Hash
  end
end

class ArraySerializer < Serializer
  def self.first_json_char
    "["
  end

  def self.object_class
    ::Array
  end
end
