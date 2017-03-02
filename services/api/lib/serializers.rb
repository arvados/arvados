require 'safe_json'

class Serializer
  def self.dump(val)
    SafeJSON.dump(val)
  end

  def self.legacy_load(s)
    val = Psych.safe_load(s)
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
    if s.nil?
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
