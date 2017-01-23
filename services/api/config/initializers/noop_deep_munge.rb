module ActionDispatch
  class Request < Rack::Request
    # This Rails method messes with valid JSON, for example turning the empty
    # array [] into 'nil'.  We don't want that, so turn it into a no-op.
    def deep_munge(hash)
      hash
    end
  end
end
