require 'test_helper'

class ResourceListTest < ActiveSupport::TestCase

  test 'links_for on a resource list that does not return links' do
    use_token :active
    results = Specimen.all
    assert_equal [], results.links_for(api_fixture('users')['active']['uuid'])
  end

  test 'links_for returns all link classes (simulated results)' do
    project_uuid = api_fixture('groups')['aproject']['uuid']
    specimen_uuid = api_fixture('specimens')['in_aproject']['uuid']
    api_response = {
      kind: 'arvados#specimenList',
      links: [{kind: 'arvados#link',
                uuid: 'zzzzz-o0j2j-asdfasdfasdfas1',
                tail_uuid: project_uuid,
                head_uuid: specimen_uuid,
                link_class: 'foo',
                name: 'Bob'},
              {kind: 'arvados#link',
                uuid: 'zzzzz-o0j2j-asdfasdfasdfas2',
                tail_uuid: project_uuid,
                head_uuid: specimen_uuid,
                link_class: nil,
                name: 'Clydesdale'}],
      items: [{kind: 'arvados#specimen',
                uuid: specimen_uuid}]
    }
    arl = ArvadosResourceList.new
    arl.results = ArvadosApiClient.new.unpack_api_response(api_response)
    assert_equal(['foo', nil],
                 (arl.
                  links_for(specimen_uuid).
                  collect { |x| x.link_class }),
                 "Expected links_for to return all link_classes")
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

  test 'get single page of items' do
    use_token :admin
    a = 0
    c = Collection.where(owner_uuid: 'zzzzz-j7d0g-0201collections').fetch_multiple_pages(false)
    c.each do
      a += 1
    end

    assert a < 201
    assert_equal c.result_limit, a
  end

end
