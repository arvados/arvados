# A Locator is used to parse and manipulate Keep locator strings.
#
# Locators obey the following syntax:
#
#   locator      ::= address hint*
#   address      ::= digest size-hint
#   digest       ::= <32 hexadecimal digits>
#   size-hint    ::= "+" [0-9]+
#   hint         ::= "+" hint-type hint-content
#   hint-type    ::= [A-Z]
#   hint-content ::= [A-Za-z0-9@_-]+
#
# Individual hints may have their own required format:
#
#   sign-hint      ::= "+A" <40 lowercase hex digits> "@" sign-timestamp
#   sign-timestamp ::= <8 lowercase hex digits>

class Locator
  def initialize(hasharg, sizearg, hintarg)
    @hash = hasharg
    @size = sizearg
    @hints = hintarg
  end

  # Locator.parse returns a Locator object parsed from the string tok.
  # Returns nil if tok could not be parsed as a valid locator.
  def self.parse(tok)
    begin
      Locator.parse!(tok)
    rescue ArgumentError => e
      nil
    end
  end

  # Locator.parse! returns a Locator object parsed from the string tok,
  # raising an ArgumentError if tok cannot be parsed.
  def self.parse!(tok)
    if tok.nil? or tok.empty?
      raise ArgumentError.new "locator is nil or empty"
    end

    m = /^([[:xdigit:]]{32})(\+([[:digit:]]+))?(\+([[:upper:]][[:alnum:]+@_-]*))?$/.match(tok.strip)
    unless m
      raise ArgumentError.new "not a valid locator #{tok}"
    end
    unless m[2]
      raise ArgumentError.new "missing size hint on #{tok}"
    end

    tokhash, _, toksize, _, trailer = m[1..5]
    tokhints = []
    if trailer
      trailer.split('+').each do |hint|
        if hint =~ /^[[:upper:]][[:alnum:]@_-]+$/
          tokhints.push(hint)
        else
          raise ArgumentError.new "unknown hint #{hint}"
        end
      end
    end

    Locator.new(tokhash, toksize, tokhints)
  end

  # Returns the signature hint supplied with this locator,
  # or nil if the locator was not signed.
  def signature
    @hints.grep(/^A/).first
  end

  # Returns an unsigned Locator.
  def without_signature
    Locator.new(@hash, @size, @hints.reject { |o| o.start_with?("A") })
  end

  def strip_hints
    Locator.new(@hash, @size, [])
  end

  def strip_hints!
    @hints = []
    self
  end

  def hash
    @hash
  end

  def size
    @size
  end

  def hints
    @hints
  end

  def to_s
    [ @hash, @size, *@hints ].join('+')
  end
end
