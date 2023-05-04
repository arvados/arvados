# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

require 'test_helper'

class CollectionsApiTest < ActionDispatch::IntegrationTest
  fixtures :all

  test "should get index" do
    get "/arvados/v1/collections",
      params: {:format => :json},
      headers: auth(:active)
    assert_response :success
    assert_equal "arvados#collectionList", json_response['kind']
  end

  test "get index with filters= (empty string)" do
    get "/arvados/v1/collections",
      params: {:format => :json, :filters => ''},
      headers: auth(:active)
    assert_response :success
    assert_equal "arvados#collectionList", json_response['kind']
  end

  test "get index with invalid filters (array of strings) responds 422" do
    get "/arvados/v1/collections",
      params: {
        :format => :json,
        :filters => ['uuid', '=', 'ad02e37b6a7f45bbe2ead3c29a109b8a+54'].to_json
      },
      headers: auth(:active)
    assert_response 422
    assert_match(/nvalid element.*not an array/, json_response['errors'].join(' '))
  end

  test "get index with invalid filters (unsearchable column) responds 422" do
    get "/arvados/v1/collections",
      params: {
        :format => :json,
        :filters => [['this_column_does_not_exist', '=', 'bogus']].to_json
      },
      headers: auth(:active)
    assert_response 422
    assert_match(/nvalid attribute/, json_response['errors'].join(' '))
  end

  test "get index with invalid filters (invalid operator) responds 422" do
    get "/arvados/v1/collections",
      params: {
        :format => :json,
        :filters => [['uuid', ':-(', 'displeased']].to_json
      },
      headers: auth(:active)
    assert_response 422
    assert_match(/nvalid operator/, json_response['errors'].join(' '))
  end

  test "get index with invalid filters (invalid operand type) responds 422" do
    get "/arvados/v1/collections",
      params: {
        :format => :json,
        :filters => [['uuid', '=', {foo: 'bar'}]].to_json
      },
      headers: auth(:active)
    assert_response 422
    assert_match(/nvalid operand type/, json_response['errors'].join(' '))
  end

  test "get index with where= (empty string)" do
    get "/arvados/v1/collections",
      params: {:format => :json, :where => ''},
      headers: auth(:active)
    assert_response :success
    assert_equal "arvados#collectionList", json_response['kind']
  end

  test "get index with select= (valid attribute)" do
    get "/arvados/v1/collections",
      params: {
        :format => :json,
        :select => ['portable_data_hash'].to_json
      },
      headers: auth(:active)
    assert_response :success
    assert json_response['items'][0].keys.include?('portable_data_hash')
    assert not(json_response['items'][0].keys.include?('uuid'))
  end

  test "get index with select= (invalid attribute) responds 422" do
    get "/arvados/v1/collections",
      params: {
        :format => :json,
        :select => ['bogus'].to_json
      },
      headers: auth(:active)
    assert_response 422
    assert_match(/Invalid attribute.*bogus/, json_response['errors'].join(' '))
  end

  test "get index with select= (invalid attribute type) responds 422" do
    get "/arvados/v1/collections",
      params: {
        :format => :json,
        :select => [['bogus']].to_json
      },
      headers: auth(:active)
    assert_response 422
    assert_match(/Invalid attribute.*bogus/, json_response['errors'].join(' '))
  end

  test "controller 404 response is json" do
    get "/arvados/v1/thingsthatdonotexist",
      params: {:format => :xml},
      headers: auth(:active)
    assert_response 404
    assert_equal 1, json_response['errors'].length
    assert_equal true, json_response['errors'][0].is_a?(String)
  end

  test "object 404 response is json" do
    get "/arvados/v1/groups/zzzzz-j7d0g-o5ba971173cup4f",
      params: {},
      headers: auth(:active)
    assert_response 404
    assert_equal 1, json_response['errors'].length
    assert_equal true, json_response['errors'][0].is_a?(String)
  end

  test "store collection as json" do
    signing_opts = {
      key: Rails.configuration.Collections.BlobSigningKey,
      api_token: api_token(:active),
    }
    signed_locator = Blob.sign_locator('bad42fa702ae3ea7d888fef11b46f450+44',
                                       signing_opts)
    post "/arvados/v1/collections",
      params: {
        format: :json,
        collection: "{\"manifest_text\":\". #{signed_locator} 0:44:md5sum.txt\\n\",\"portable_data_hash\":\"ad02e37b6a7f45bbe2ead3c29a109b8a+54\"}"
      },
      headers: auth(:active)
    assert_response 200
    assert_equal 'ad02e37b6a7f45bbe2ead3c29a109b8a+54', json_response['portable_data_hash']
  end

  test "store collection with manifest_text only" do
    signing_opts = {
      key: Rails.configuration.Collections.BlobSigningKey,
      api_token: api_token(:active),
    }
    signed_locator = Blob.sign_locator('bad42fa702ae3ea7d888fef11b46f450+44',
                                       signing_opts)
    post "/arvados/v1/collections",
      params: {
        format: :json,
        collection: "{\"manifest_text\":\". #{signed_locator} 0:44:md5sum.txt\\n\"}"
      },
      headers: auth(:active)
    assert_response 200
    assert_equal 'ad02e37b6a7f45bbe2ead3c29a109b8a+54', json_response['portable_data_hash']
  end

  test "store collection then update name" do
    signing_opts = {
      key: Rails.configuration.Collections.BlobSigningKey,
      api_token: api_token(:active),
    }
    signed_locator = Blob.sign_locator('bad42fa702ae3ea7d888fef11b46f450+44',
                                       signing_opts)
    post "/arvados/v1/collections",
      params: {
        format: :json,
        collection: "{\"manifest_text\":\". #{signed_locator} 0:44:md5sum.txt\\n\",\"portable_data_hash\":\"ad02e37b6a7f45bbe2ead3c29a109b8a+54\"}"
      },
      headers: auth(:active)
    assert_response 200
    assert_equal 'ad02e37b6a7f45bbe2ead3c29a109b8a+54', json_response['portable_data_hash']

    put "/arvados/v1/collections/#{json_response['uuid']}",
      params: {
        format: :json,
        collection: { name: "a name" }
      },
      headers: auth(:active)

    assert_response 200
    assert_equal 'ad02e37b6a7f45bbe2ead3c29a109b8a+54', json_response['portable_data_hash']
    assert_equal 'a name', json_response['name']

    get "/arvados/v1/collections/#{json_response['uuid']}",
      params: {format: :json},
      headers: auth(:active)

    assert_response 200
    assert_equal 'ad02e37b6a7f45bbe2ead3c29a109b8a+54', json_response['portable_data_hash']
    assert_equal 'a name', json_response['name']
  end

  test "update description for a collection, and search for that description" do
    collection = collections(:multilevel_collection_1)

    # update collection's description
    put "/arvados/v1/collections/#{collection['uuid']}",
      params: {
        format: :json,
        collection: { description: "something specific" }
      },
      headers: auth(:active)
    assert_response :success
    assert_equal 'something specific', json_response['description']

    # get the collection and verify newly added description
    get "/arvados/v1/collections/#{collection['uuid']}",
      params: {format: :json},
      headers: auth(:active)
    assert_response 200
    assert_equal 'something specific', json_response['description']

    # search
    search_using_filter 'specific', 1
    search_using_filter 'not specific enough', 0
  end

  test "create collection, update manifest, and search with filename" do
    # create collection
    signed_manifest = Collection.sign_manifest_only_for_tests(". bad42fa702ae3ea7d888fef11b46f450+44 0:44:my_test_file.txt\n", api_token(:active))
    post "/arvados/v1/collections",
      params: {
        format: :json,
        collection: {manifest_text: signed_manifest}.to_json,
      },
      headers: auth(:active)
    assert_response :success
    assert_equal true, json_response['manifest_text'].include?('my_test_file.txt')
    assert_includes json_response['manifest_text'], 'my_test_file.txt'

    created = json_response

    # search using the filename
    search_using_filter 'my_test_file.txt', 1

    # update the collection's manifest text
    signed_manifest = Collection.sign_manifest_only_for_tests(". bad42fa702ae3ea7d888fef11b46f450+44 0:44:my_updated_test_file.txt\n", api_token(:active))
    put "/arvados/v1/collections/#{created['uuid']}",
      params: {
        format: :json,
        collection: {manifest_text: signed_manifest}.to_json,
      },
      headers: auth(:active)
    assert_response :success
    assert_equal created['uuid'], json_response['uuid']
    assert_includes json_response['manifest_text'], 'my_updated_test_file.txt'
    assert_not_includes json_response['manifest_text'], 'my_test_file.txt'

    # search using the new filename
    search_using_filter 'my_updated_test_file.txt', 1
    search_using_filter 'my_test_file.txt', 0
    search_using_filter 'there_is_no_such_file.txt', 0
  end

  def search_using_filter search_filter, expected_items
    get '/arvados/v1/collections',
      params: {:filters => [['any', 'ilike', "%#{search_filter}%"]].to_json},
      headers: auth(:active)
    assert_response :success
    response_items = json_response['items']
    assert_not_nil response_items
    if expected_items == 0
      assert_empty response_items
    else
      refute_empty response_items
      first_item = response_items.first
      assert_not_nil first_item
    end
  end

  [
    ["false", false],
    ["0", false],
    ["true", true],
    ["1", true]
  ].each do |param, truthiness|
    test "include_trash=#{param.inspect} param JSON-encoded should be interpreted as include_trash=#{truthiness}" do
      expired_col = collections(:expired_collection)
      assert expired_col.is_trashed
      # Try #index first
      post "/arvados/v1/collections",
          params: {
            :_method => 'GET',
            :include_trash => param,
            :filters => [['uuid', '=', expired_col.uuid]].to_json
          },
          headers: auth(:active)
      assert_response :success
      assert_not_nil json_response['items']
      assert_equal truthiness, json_response['items'].collect {|c| c['uuid']}.include?(expired_col.uuid)
      # Try #show next
      post "/arvados/v1/collections/#{expired_col.uuid}",
        params: {
          :_method => 'GET',
          :include_trash => param,
        },
        headers: auth(:active)
      if truthiness
        assert_response :success
      else
        assert_response 404
      end
    end
  end

  [
    ["false", false],
    ["0", false],
    ["true", true],
    ["1", true]
  ].each do |param, truthiness|
    test "include_trash=#{param.inspect} param encoding via query string should be interpreted as include_trash=#{truthiness}" do
      expired_col = collections(:expired_collection)
      assert expired_col.is_trashed
      # Try #index first
      get("/arvados/v1/collections?include_trash=#{param}&filters=#{[['uuid','=',expired_col.uuid]].to_json}",
          headers: auth(:active))
      assert_response :success
      assert_not_nil json_response['items']
      assert_equal truthiness, json_response['items'].collect {|c| c['uuid']}.include?(expired_col.uuid)
      # Try #show next
      get("/arvados/v1/collections/#{expired_col.uuid}?include_trash=#{param}",
        headers: auth(:active))
      if truthiness
        assert_response :success
      else
        assert_response 404
      end
    end
  end

  [
    ["false", false],
    ["0", false],
    ["true", true],
    ["1", true]
  ].each do |param, truthiness|
    test "include_trash=#{param.inspect} form-encoded param should be interpreted as include_trash=#{truthiness}" do
      expired_col = collections(:expired_collection)
      assert expired_col.is_trashed
      params = [
        ['_method', 'GET'],
        ['include_trash', param],
        ['filters', [['uuid','=',expired_col.uuid]].to_json],
      ]
      # Try #index first
      post "/arvados/v1/collections",
        params: URI.encode_www_form(params),
        headers: {
          "Content-type" => "application/x-www-form-urlencoded"
        }.update(auth(:active))
      assert_response :success
      assert_not_nil json_response['items']
      assert_equal truthiness, json_response['items'].collect {|c| c['uuid']}.include?(expired_col.uuid)
      # Try #show next
      post "/arvados/v1/collections/#{expired_col.uuid}",
        params: URI.encode_www_form([['_method', 'GET'],['include_trash', param]]),
        headers: {
          "Content-type" => "application/x-www-form-urlencoded"
        }.update(auth(:active))
      if truthiness
        assert_response :success
      else
        assert_response 404
      end
    end
  end

  test "create and get collection with properties" do
    # create collection to be searched for
    signed_manifest = Collection.sign_manifest_only_for_tests(". bad42fa702ae3ea7d888fef11b46f450+44 0:44:my_test_file.txt\n", api_token(:active))
    post "/arvados/v1/collections",
      params: {
        format: :json,
        collection: {manifest_text: signed_manifest}.to_json,
      },
      headers: auth(:active)
    assert_response 200
    assert_not_nil json_response['uuid']
    assert_not_nil json_response['properties']
    assert_empty json_response['properties']

    # update collection's properties
    put "/arvados/v1/collections/#{json_response['uuid']}",
      params: {
        format: :json,
        collection: { properties: {'property_1' => 'value_1'} }
      },
      headers: auth(:active)
    assert_response :success
    assert_equal Hash, json_response['properties'].class, 'Collection properties attribute should be of type hash'
    assert_equal 'value_1', json_response['properties']['property_1']
  end

  test "create collection and update it with json encoded hash properties" do
    # create collection to be searched for
    signed_manifest = Collection.sign_manifest_only_for_tests(". bad42fa702ae3ea7d888fef11b46f450+44 0:44:my_test_file.txt\n", api_token(:active))
    post "/arvados/v1/collections",
      params: {
        format: :json,
        collection: {manifest_text: signed_manifest}.to_json,
      },
      headers: auth(:active)
    assert_response 200
    assert_not_nil json_response['uuid']
    assert_not_nil json_response['properties']
    assert_empty json_response['properties']

    # update collection's properties
    put "/arvados/v1/collections/#{json_response['uuid']}",
      params: {
        format: :json,
        collection: {
          properties: "{\"property_1\":\"value_1\"}"
        }
      },
      headers: auth(:active)
    assert_response :success
    assert_equal Hash, json_response['properties'].class, 'Collection properties attribute should be of type hash'
    assert_equal 'value_1', json_response['properties']['property_1']
  end

  test "update collection with versioning enabled and using preserve_version" do
    Rails.configuration.Collections.CollectionVersioning = true
    Rails.configuration.Collections.PreserveVersionIfIdle = -1 # Disable auto versioning

    signed_manifest = Collection.sign_manifest_only_for_tests(". bad42fa702ae3ea7d888fef11b46f450+44 0:44:my_test_file.txt\n", api_token(:active))
    post "/arvados/v1/collections",
      params: {
        format: :json,
        collection: {
          name: 'Test collection',
          manifest_text: signed_manifest,
        }.to_json,
      },
      headers: auth(:active)
    assert_response 200
    assert_not_nil json_response['uuid']
    assert_equal 1, json_response['version']
    assert_equal false, json_response['preserve_version']

    # Versionable update including preserve_version=true should create a new
    # version that will also be persisted.
    put "/arvados/v1/collections/#{json_response['uuid']}",
      params: {
        format: :json,
        collection: {
          name: 'Test collection v2',
          preserve_version: true,
        }.to_json,
      },
      headers: auth(:active)
    assert_response 200
    assert_equal 2, json_response['version']
    assert_equal true, json_response['preserve_version']

    # 2nd versionable update including preserve_version=true should create a new
    # version that will also be persisted.
    put "/arvados/v1/collections/#{json_response['uuid']}",
      params: {
        format: :json,
        collection: {
          name: 'Test collection v3',
          preserve_version: true,
        }.to_json,
      },
      headers: auth(:active)
    assert_response 200
    assert_equal 3, json_response['version']
    assert_equal true, json_response['preserve_version']

    # 3rd versionable update without including preserve_version should create a new
    # version that will have its preserve_version attr reset to false.
    put "/arvados/v1/collections/#{json_response['uuid']}",
      params: {
        format: :json,
        collection: {
          name: 'Test collection v4',
        }.to_json,
      },
      headers: auth(:active)
    assert_response 200
    assert_equal 4, json_response['version']
    assert_equal false, json_response['preserve_version']

    # 4th versionable update without including preserve_version=true should NOT
    # create a new version.
    put "/arvados/v1/collections/#{json_response['uuid']}",
      params: {
        format: :json,
        collection: {
          name: 'Test collection v5?',
        }.to_json,
      },
      headers: auth(:active)
    assert_response 200
    assert_equal 4, json_response['version']
    assert_equal false, json_response['preserve_version']
  end
end
