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

  test 'get limited items, limit % page_size != 0' do
    skip "Requires server MAX_LIMIT < 200 which is not currently the default"

    use_token :admin
    max_page_size = Collection.
      where(owner_uuid: 'zzzzz-j7d0g-0201collections').
      limit(1000000000).
      fetch_multiple_pages(false).
      count
    # Conditions necessary for this test to be valid:
    assert_operator 200, :>, max_page_size
    assert_operator 1, :<, max_page_size
    # Verify that the server really sends max_page_size when asked for max_page_size+1
    assert_equal max_page_size, Collection.
      where(owner_uuid: 'zzzzz-j7d0g-0201collections').
      limit(max_page_size+1).
      fetch_multiple_pages(false).
      results.
      count
    # Now that we know the max_page_size+1 is in the middle of page 2,
    # make sure #each returns page 1 and only the requested part of
    # page 2.
    a = 0
    saw_uuid = {}
    Collection.where(owner_uuid: 'zzzzz-j7d0g-0201collections').limit(max_page_size+1).each do |item|
      a += 1
      saw_uuid[item.uuid] = true
    end
    assert_equal max_page_size+1, a
    # Ensure no overlap between pages
    assert_equal max_page_size+1, saw_uuid.size
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

  test 'get empty set' do
    use_token :admin
    c = Collection.
      where(owner_uuid: 'doesn-texis-tdoesntexistdoe').
      fetch_multiple_pages(false)
    # Important: check c.result_offset before calling c.results here.
    assert_equal 0, c.result_offset
    assert_equal 0, c.items_available
    assert_empty c.results
  end

end
