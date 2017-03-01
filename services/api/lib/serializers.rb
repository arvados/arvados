require 'safe_json'

class HashSerializer
  def self.load(s)
    if s.nil?
      {}
    elsif s[0] == "{"
      SafeJSON.load(s)
    elsif s[0..2] == "---"
      Psych.safe_load(s)
    else
      raise "invalid serialized data #{s[0..5].inspect}"
    end
  end
  def self.dump(h)
    SafeJSON.dump(h)
  end
  def self.object_class
    ::Hash
  end
end

class ArraySerializer
  def self.load(s)
    if s.nil?
      []
    elsif s[0] == "["
      SafeJSON.load(s)
    elsif s[0..2] == "---"
      Psych.safe_load(s)
    else
      raise "invalid serialized data #{s[0..5].inspect}"
    end
  end
  def self.dump(a)
    SafeJSON.dump(a)
  end
  def self.object_class
    ::Array
  end
end
