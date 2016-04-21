require 'test_helper'

class BlobTest < ActiveSupport::TestCase
  @@api_token = rand(2**512).to_s(36)[0..49]
  @@key = rand(2**2048).to_s(36)
  @@blob_data = 'foo'
  @@blob_locator = Digest::MD5.hexdigest(@@blob_data) +
    '+' + @@blob_data.size.to_s

  @@known_locator = 'acbd18db4cc2f85cedef654fccc4a4d8+3'
  @@known_token = 'hocfupkn2pjhrpgp2vxv8rsku7tvtx49arbc9s4bvu7p7wxqvk'
  @@known_key = '13u9fkuccnboeewr0ne3mvapk28epf68a3bhj9q8sb4l6e4e5mkk' +
    'p6nhj2mmpscgu1zze5h5enydxfe3j215024u16ij4hjaiqs5u4pzsl3nczmaoxnc' +
    'ljkm4875xqn4xv058koz3vkptmzhyheiy6wzevzjmdvxhvcqsvr5abhl15c2d4o4' +
    'jhl0s91lojy1mtrzqqvprqcverls0xvy9vai9t1l1lvvazpuadafm71jl4mrwq2y' +
    'gokee3eamvjy8qq1fvy238838enjmy5wzy2md7yvsitp5vztft6j4q866efym7e6' +
    'vu5wm9fpnwjyxfldw3vbo01mgjs75rgo7qioh8z8ij7jpyp8508okhgbbex3ceei' +
    '786u5rw2a9gx743dj3fgq2irk'
  @@known_signed_locator = 'acbd18db4cc2f85cedef654fccc4a4d8+3' +
    '+A89118b78732c33104a4d6231e8b5a5fa1e4301e3@7fffffff'

  test 'generate predictable invincible signature' do
    signed = Blob.sign_locator @@known_locator, {
      api_token: @@known_token,
      key: @@known_key,
      expire: 0x7fffffff,
    }
    assert_equal @@known_signed_locator, signed
  end

  test 'verify predictable invincible signature' do
    assert_equal true, Blob.verify_signature!(@@known_signed_locator,
                                              api_token: @@known_token,
                                              key: @@known_key)
  end

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

  test 'signature changes when ttl changes' do
    signed = Blob.sign_locator @@known_locator, {
      api_token: @@known_token,
      key: @@known_key,
      expire: 0x7fffffff,
    }

    original_ttl = Rails.configuration.blob_signature_ttl
    Rails.configuration.blob_signature_ttl = original_ttl*2
    signed2 = Blob.sign_locator @@known_locator, {
      api_token: @@known_token,
      key: @@known_key,
      expire: 0x7fffffff,
    }
    Rails.configuration.blob_signature_ttl = original_ttl

    assert_not_equal signed, signed2
  end
end
