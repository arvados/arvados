require 'test_helper'

class ArvadosBaseTest < ActiveSupport::TestCase
  test '#save does not send unchanged string attributes' do
    use_token :active do
      fixture = api_fixture("collections")["foo_collection_in_aproject"]
      c = Collection.find(fixture['uuid'])

      new_name = 'name changed during test'

      got_query = nil
      stub_api_calls
      stub_api_client.expects(:post).with do |url, query, opts={}|
        got_query = query
        true
      end.returns fake_api_response('{}', 200, {})
      c.name = new_name
      c.save

      updates = JSON.parse got_query['collection']
      assert_equal updates['name'], new_name
      refute_includes updates, 'description'
      refute_includes updates, 'manifest_text'
    end
  end

  test '#save does not send unchanged attributes missing because of select' do
    use_token :active do
      fixture = api_fixture("collections")["foo_collection_in_aproject"]
      c = Collection.
        filter([['uuid','=',fixture['uuid']]]).
        select(['uuid']).
        first
      assert_equal nil, c.properties

      got_query = nil
      stub_api_calls
      stub_api_client.expects(:post).with do |url, query, opts={}|
        got_query = query
        true
      end.returns fake_api_response('{}', 200, {})
      c.name = 'foo'
      c.save

      updates = JSON.parse got_query['collection']
      assert_includes updates, 'name'
      refute_includes updates, 'description'
      refute_includes updates, 'properties'
    end
  end

  [false,
   {},
   {'foo' => 'bar'},
  ].each do |init_props|
    test "#save sends serialized attributes if changed from #{init_props}" do
      use_token :active do
        fixture = api_fixture("collections")["foo_collection_in_aproject"]
        c = Collection.find(fixture['uuid'])

        if init_props
          c.properties = init_props if init_props
          c.save!
        end

        got_query = nil
        stub_api_calls
        stub_api_client.expects(:post).with do |url, query, opts={}|
          got_query = query
          true
        end.returns fake_api_response('{"etag":"fake","uuid":"fake"}', 200, {})

        c.properties['baz'] = 'qux'
        c.save!

        updates = JSON.parse got_query['collection']
        assert_includes updates, 'properties'
      end
    end
  end
end
