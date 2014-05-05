require 'test_helper'

class ResourceListTest < ActiveSupport::TestCase

  test 'links_for on a resource list that does not return links' do
    use_token :active
    results = Specimen.all
    assert_equal [], results.links_for(api_fixture('users')['active']['uuid'])
  end

  test 'links_for on non-empty resource list' do
    use_token :active
    results = Group.find(api_fixture('groups')['afolder']['uuid']).contents(include_linked: true)
    assert_equal [], results.links_for(api_fixture('users')['active']['uuid'])
    assert_equal [], results.links_for(api_fixture('jobs')['running_cancelled']['uuid'])
    assert_equal [], results.links_for(api_fixture('jobs')['running']['uuid'], 'bogus-link-class')
    assert_equal true, results.links_for(api_fixture('jobs')['running']['uuid'], 'name').any?
  end

  test 'links_for returns all link classes (simulated results)' do
    folder_uuid = api_fixture('groups')['afolder']['uuid']
    specimen_uuid = api_fixture('specimens')['in_afolder']['uuid']
    api_response = {
      kind: 'arvados#specimenList',
      links: [{kind: 'arvados#link',
                uuid: 'zzzzz-o0j2j-asdfasdfasdfas0',
                tail_uuid: folder_uuid,
                head_uuid: specimen_uuid,
                link_class: 'name',
                name: 'Alice'},
              {kind: 'arvados#link',
                uuid: 'zzzzz-o0j2j-asdfasdfasdfas1',
                tail_uuid: folder_uuid,
                head_uuid: specimen_uuid,
                link_class: 'foo',
                name: 'Bob'},
              {kind: 'arvados#link',
                uuid: 'zzzzz-o0j2j-asdfasdfasdfas2',
                tail_uuid: folder_uuid,
                head_uuid: specimen_uuid,
                link_class: nil,
                name: 'Clydesdale'}],
      items: [{kind: 'arvados#specimen',
                uuid: specimen_uuid}]
    }
    results = ArvadosApiClient.new.unpack_api_response(api_response)
    assert_equal(['name', 'foo', nil],
                 (results.
                  links_for(specimen_uuid).
                  collect { |x| x.link_class }),
                 "Expected links_for to return all link_classes")
  end

end
