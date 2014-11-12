require 'test_helper'

class ResourceListTest < ActiveSupport::TestCase

  test 'links_for on a resource list that does not return links' do
    use_token :active
    results = Specimen.all
    assert_equal [], results.links_for(api_fixture('users')['active']['uuid'])
  end

  test 'get all items by default' do
    use_token :admin
    a = 0
    Collection.where(owner_uuid: 'zzzzz-j7d0g-0201collections').each do
      a += 1
    end
    assert_equal 201, a
  end

  test 'prefetch all items' do
    use_token :admin
    a = 0
    Collection.where(owner_uuid: 'zzzzz-j7d0g-0201collections').each do
      a += 1
    end
    assert_equal 201, a
  end

  test 'get limited items' do
    use_token :admin
    a = 0
    Collection.where(owner_uuid: 'zzzzz-j7d0g-0201collections').limit(51).each do
      a += 1
    end
    assert_equal 51, a
  end

  test 'get limited items more than default page size' do
    use_token :admin
    a = 0
    Collection.where(owner_uuid: 'zzzzz-j7d0g-0201collections').limit(110).each do
      a += 1
    end
    assert_equal 110, a
  end

  test 'get single page of items' do
    use_token :admin
    a = 0
    c = Collection.where(owner_uuid: 'zzzzz-j7d0g-0201collections').fetch_multiple_pages(false)
    c.each do
      a += 1
    end

    assert_operator a, :<, 201
    assert_equal c.result_limit, a
  end

end
