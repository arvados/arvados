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

end
