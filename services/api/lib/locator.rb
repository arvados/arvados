class Locator
  # This regex will match a word that appears to be a locator.
  @@pattern = %r![[:xdigit:]]{32}(\+[[:digit:]]+)?(\+\S+)?!

  def initialize(hasharg, sizearg, optarg)
    @hash = hasharg
    @size = sizearg
    @options = optarg
  end

  def self.parse(tok)
    Locator.parse!(tok) rescue nil
  end

  def self.parse!(tok)
    m = /^([[:xdigit:]]{32})(\+([[:digit:]]+))?(\+.*)?$/.match(tok)
    unless m
      raise ArgumentError.new "could not parse #{tok}"
    end

    tokhash, _, toksize, trailer = m[1..4]
    tokopts = []
    while m = /^\+[[:upper:]][^\s+]+/.match(trailer)
      opt = m.to_s
      if opt =~ /^\+A[[:xdigit:]]+@[[:xdigit:]]{8}$/ or
          opt =~ /\+K@[[:alnum:]]+$/
        tokopts.push(opt)
        trailer = m.post_match
      else
        raise ArgumentError.new "unknown option #{opt}"
      end
    end
    if trailer and !trailer.empty?
      raise ArgumentError.new "unrecognized trailing chars #{trailer}"
    end

    Locator.new(tokhash, toksize, tokopts)
  end

  def signature
    @options.grep(/^\+A/).first
  end

  def without_signature
    Locator.new(@hash, @size, @options.reject { |o| o.start_with?("+A") })
  end

  def hash
    @hash
  end

  def size
    @size
  end

  def options
    @options
  end

  def to_s
    [ @hash + "+", @size, *@options].join
  end
end
