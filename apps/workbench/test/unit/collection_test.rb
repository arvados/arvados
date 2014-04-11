require 'test_helper'

class CollectionTest < ActiveSupport::TestCase
  test 'recognize empty blob locator' do
    ['d41d8cd98f00b204e9800998ecf8427e+0',
     'd41d8cd98f00b204e9800998ecf8427e',
     'd41d8cd98f00b204e9800998ecf8427e+0+Xyzzy'].each do |x|
      assert_equal true, Collection.is_empty_blob_locator?(x)
    end
    ['d41d8cd98f00b204e9800998ecf8427e0',
     'acbd18db4cc2f85cedef654fccc4a4d8+3',
     'acbd18db4cc2f85cedef654fccc4a4d8+0'].each do |x|
      assert_equal false, Collection.is_empty_blob_locator?(x)
    end
  end
end
