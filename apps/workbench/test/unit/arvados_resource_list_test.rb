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

end
