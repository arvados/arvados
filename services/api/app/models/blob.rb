class Blob
  class InvalidSignatureError < StandardError
  end

  def self.sign_locator blob_locator, opts
    # We only use the hash portion for signatures.
    blob_hash = blob_locator.split('+').first

    # Generate an expiry timestamp (seconds since epoch, base 16)
    timestamp = (Time.now.to_i + (opts[:ttl] || 600)).to_s(16)
    # => "53163cb4"

    # Generate a signature.
    signature =
      OpenSSL::HMAC.hexdigest('sha1', opts[:key],
                              [blob_hash,
                               opts[:api_token],
                               timestamp].join('@'))

    blob_locator + '+A' + signature + '@' + timestamp
  end

  def self.verify_signature *args
    begin
      self.verify_signature! *args
      true
    rescue Blob::InvalidSignatureError
      false
    end
  end

  def self.verify_signature! signed_blob_locator, opts
    blob_hash = signed_blob_locator.split('+').first
    given_signature, timestamp = signed_blob_locator.
      split('+A').last.
      split('+').first.
      split('@')

    if !timestamp
      raise Blob::InvalidSignatureError.new 'No signature provided.'
    end
    if !timestamp.match /^[\da-f]+$/
      raise Blob::InvalidSignatureError.new 'Timestamp is not a base16 number.'
    end
    if timestamp.to_i(16) < Time.now.to_i
      raise Blob::InvalidSignatureError.new 'Signature expiry time has passed.'
    end

    my_signature =
      OpenSSL::HMAC.hexdigest('sha1', opts[:key],
                              [blob_hash,
                               opts[:api_token],
                               timestamp].join('@'))
    if my_signature != given_signature
      raise Blob::InvalidSignatureError.new 'Signature is invalid.'
    end

    true
  end
end
