class Blob

  # In order to get a Blob from Keep, you have to prove either
  # [a] you have recently written it to Keep yourself, or
  # [b] apiserver has recently decided that you should be able to read it
  #
  # To ensure that the requestor of a blob is authorized to read it,
  # Keep requires clients to timestamp the blob locator with an expiry
  # time, and to sign the timestamped locator with their API token.
  #
  # A signed blob locator has the form:
  #     locator_hash +A blob_signature @ timestamp
  # where the timestamp is a Unix time expressed as a hexadecimal value,
  # and the blob_signature is the signed locator_hash + API token + timestamp.
  # 
  class InvalidSignatureError < StandardError
  end

  # Blob.sign_locator: return a signed and timestamped blob locator.
  #
  # The 'opts' argument should include:
  #   [required] :key       - the Arvados server-side blobstore key
  #   [required] :api_token - user's API token
  #   [optional] :ttl       - number of seconds before this request expires
  #
  def self.sign_locator blob_locator, opts
    # We only use the hash portion for signatures.
    blob_hash = blob_locator.split('+').first

    # Generate an expiry timestamp (seconds since epoch, base 16)
    timestamp = (Time.now.to_i + (opts[:ttl] || 600)).to_s(16)
    # => "53163cb4"

    # Generate a signature.
    signature =
      generate_signature opts[:key], blob_hash, opts[:api_token], timestamp

    blob_locator + '+A' + signature + '@' + timestamp
  end

  # Blob.verify_signature
  #   Safely verify the signature on a blob locator.
  #   Return value: true if the locator has a valid signature, false otherwise
  #   Arguments: signed_blob_locator, opts
  #
  def self.verify_signature *args
    begin
      self.verify_signature! *args
      true
    rescue Blob::InvalidSignatureError
      false
    end
  end

  # Blob.verify_signature!
  #   Verify the signature on a blob locator.
  #   Return value: true if the locator has a valid signature
  #   Arguments: signed_blob_locator, opts
  #   Exceptions:
  #     Blob::InvalidSignatureError if the blob locator does not include a
  #     valid signature
  #
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
      generate_signature opts[:key], blob_hash, opts[:api_token], timestamp

    if my_signature != given_signature
      raise Blob::InvalidSignatureError.new 'Signature is invalid.'
    end

    true
  end

  def self.generate_signature key, blob_hash, api_token, timestamp
    OpenSSL::HMAC.hexdigest('sha1', key,
                            [blob_hash,
                             api_token,
                             timestamp].join('@'))
  end
end
