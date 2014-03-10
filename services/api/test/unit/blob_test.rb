require 'test_helper'

class BlobTest < ActiveSupport::TestCase
  @@api_token = rand(2**512).to_s(36)[0..49]
  @@key = rand(2**2048).to_s(36)
  @@blob_data = 'foo'
  @@blob_locator = Digest::MD5.hexdigest(@@blob_data) +
    '+' + @@blob_data.size.to_s

  test 'correct' do
    signed = Blob.sign_locator @@blob_locator, api_token: @@api_token, key: @@key
    assert_equal true, Blob.verify_signature!(signed, api_token: @@api_token, key: @@key)
  end

  test 'expired' do
    signed = Blob.sign_locator @@blob_locator, api_token: @@api_token, key: @@key, ttl: -1
    assert_raise Blob::InvalidSignatureError do
      Blob.verify_signature!(signed, api_token: @@api_token, key: @@key)
    end
  end

  test 'expired, but no raise' do
    signed = Blob.sign_locator @@blob_locator, api_token: @@api_token, key: @@key, ttl: -1
    assert_equal false, Blob.verify_signature(signed,
                                              api_token: @@api_token,
                                              key: @@key)
  end

  test 'bogus, wrong block hash' do
    signed = Blob.sign_locator @@blob_locator, api_token: @@api_token, key: @@key
    assert_raise Blob::InvalidSignatureError do
      Blob.verify_signature!(signed.sub('acbd','abcd'), api_token: @@api_token, key: @@key)
    end
  end

  test 'bogus, expired' do
    signed = 'acbd18db4cc2f85cedef654fccc4a4d8+3+Aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa@531641bf'
    assert_raises Blob::InvalidSignatureError do
      Blob.verify_signature!(signed, api_token: @@api_token, key: @@key)
    end
  end

  test 'bogus, wrong key' do
    signed = Blob.sign_locator(@@blob_locator,
                               api_token: @@api_token,
                               key: (@@key+'x'))
    assert_raise Blob::InvalidSignatureError do
      Blob.verify_signature!(signed, api_token: @@api_token, key: @@key)
    end
  end

  test 'bogus, wrong api token' do
    signed = Blob.sign_locator(@@blob_locator,
                               api_token: @@api_token.reverse,
                               key: @@key)
    assert_raise Blob::InvalidSignatureError do
      Blob.verify_signature!(signed, api_token: @@api_token, key: @@key)
    end
  end

  test 'bogus, signature format 1' do
    signed = 'acbd18db4cc2f85cedef654fccc4a4d8+3+Aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa@'
    assert_raise Blob::InvalidSignatureError do
      Blob.verify_signature!(signed, api_token: @@api_token, key: @@key)
    end
  end

  test 'bogus, signature format 2' do
    signed = 'acbd18db4cc2f85cedef654fccc4a4d8+3+A@531641bf'
    assert_raise Blob::InvalidSignatureError do
      Blob.verify_signature!(signed, api_token: @@api_token, key: @@key)
    end
  end

  test 'bogus, signature format 3' do
    signed = 'acbd18db4cc2f85cedef654fccc4a4d8+3+Axyzzy@531641bf'
    assert_raise Blob::InvalidSignatureError do
      Blob.verify_signature!(signed, api_token: @@api_token, key: @@key)
    end
  end

  test 'bogus, timestamp format' do
    signed = 'acbd18db4cc2f85cedef654fccc4a4d8+3+Aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa@xyzzy'
    assert_raise Blob::InvalidSignatureError do
      Blob.verify_signature!(signed, api_token: @@api_token, key: @@key)
    end
  end

  test 'no signature at all' do
    assert_raise Blob::InvalidSignatureError do
      Blob.verify_signature!(@@blob_locator, api_token: @@api_token, key: @@key)
    end
  end
end
