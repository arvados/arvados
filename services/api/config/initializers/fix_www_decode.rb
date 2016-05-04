module URI
  if Gem::Version.new(RUBY_VERSION) < Gem::Version.new('2.2')
    # Rack uses the standard library method URI.decode_www_form_component to
    # process parameters.  This method first validates the string with a
    # regular expression, and then decodes it using another regular expression.
    # Ruby 2.1 and earlier has a bug is in the validation; the regular
    # expression that is used generates many backtracking points, which results
    # in exponential memory growth when matching large strings.  The fix is to
    # monkey-patch the version of the method from Ruby 2.2 which checks that
    # the string is not invalid instead of checking it is valid.
    def self.decode_www_form_component(str, enc=Encoding::UTF_8)
      raise ArgumentError, "invalid %-encoding (#{str})" if /%(?!\h\h)/ =~ str
      str.b.gsub(/\+|%\h\h/, TBLDECWWWCOMP_).force_encoding(enc)
    end
  end
end
