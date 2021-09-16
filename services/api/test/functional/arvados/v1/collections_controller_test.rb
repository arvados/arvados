# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

require 'test_helper'

class Arvados::V1::CollectionsControllerTest < ActionController::TestCase
  include DbCurrentTime

  PERM_TOKEN_RE = /\+A[[:xdigit:]]+@[[:xdigit:]]{8}\b/

  def permit_unsigned_manifests isok=true
    # Set security model for the life of a test.
    Rails.configuration.Collections.BlobSigning = !isok
  end

  def assert_signed_manifest manifest_text, label='', token: false
    assert_not_nil manifest_text, "#{label} manifest_text was nil"
    manifest_text.scan(/ [[:xdigit:]]{32}\S*/) do |tok|
      assert_match(PERM_TOKEN_RE, tok,
                   "Locator in #{label} manifest_text was not signed")
      if token
        bare = tok.gsub(/\+A[^\+]*/, '').sub(/^ /, '')
        exp = tok[/\+A[[:xdigit:]]+@([[:xdigit:]]+)/, 1].to_i(16)
        sig = Blob.sign_locator(
          bare,
          key: Rails.configuration.Collections.BlobSigningKey,
          expire: exp,
          api_token: token)[/\+A[^\+]*/, 0]
        assert_includes tok, sig
      end
    end
  end

  def assert_unsigned_manifest txt, label=''
    assert_not_nil(txt, "#{label} unsigned_manifest_text was nil")
    locs = 0
    txt.scan(/ [[:xdigit:]]{32}\S*/) do |tok|
      locs += 1
      refute_match(PERM_TOKEN_RE, tok,
                   "Locator in #{label} unsigned_manifest_text was signed: #{tok}")
    end
    return locs
  end

  test "should get index" do
    authorize_with :active
    get :index
    assert_response :success
    assert(assigns(:objects).andand.any?, "no Collections returned in index")
    refute(json_response["items"].any? { |c| c.has_key?("manifest_text") },
           "basic Collections index included manifest_text")
    refute(json_response["items"].any? { |c| c["uuid"] == collections(:collection_owned_by_active_past_version_1).uuid },
           "basic Collections index included past version")
  end

  test "get index with include_old_versions" do
    authorize_with :active
    get :index, params: {
      include_old_versions: true
    }
    assert_response :success
    assert(assigns(:objects).andand.any?, "no Collections returned in index")
    assert(json_response["items"].any? { |c| c["uuid"] == collections(:collection_owned_by_active_past_version_1).uuid },
           "past version not included on index")
  end

  test "collections.get returns unsigned locators, and no unsigned_manifest_text" do
    permit_unsigned_manifests
    authorize_with :active
    get :show, params: {id: collections(:foo_file).uuid}
    assert_response :success
    assert_unsigned_manifest json_response["manifest_text"], 'foo_file'
    refute_includes json_response, 'unsigned_manifest_text'
  end

  ['v1token', 'v2token'].each do |token_method|
    test "signatures with #{token_method} are accepted" do
      token = api_client_authorizations(:active).send(token_method)
      signed = Blob.sign_locator(
        'acbd18db4cc2f85cedef654fccc4a4d8+3',
        key: Rails.configuration.Collections.BlobSigningKey,
        api_token: token)
      authorize_with_token token
      put :update, params: {
            id: collections(:collection_owned_by_active).uuid,
            collection: {
              manifest_text: ". #{signed} 0:3:foo.txt\n",
            },
          }
      assert_response :success
      assert_unsigned_manifest json_response['manifest_text'], 'updated'
    end
  end

  test "index with manifest_text selected returns unsigned locators" do
    columns = %w(uuid owner_uuid manifest_text)
    authorize_with :active
    get :index, params: {select: columns}
    assert_response :success
    assert(assigns(:objects).andand.any?,
           "no Collections returned for index with columns selected")
    json_response["items"].each do |coll|
      assert_equal(coll.keys - ['kind'], columns,
                   "Collections index did not respect selected columns")
      assert_unsigned_manifest coll['manifest_text'], coll['uuid']
    end
  end

  test "index with unsigned_manifest_text selected returns only unsigned locators" do
    authorize_with :active
    get :index, params: {select: ['unsigned_manifest_text']}
    assert_response :success
    assert_operator json_response["items"].count, :>, 0
    locs = 0
    json_response["items"].each do |coll|
      assert_equal(coll.keys - ['kind'], ['unsigned_manifest_text'],
                   "Collections index did not respect selected columns")
      assert_nil coll['manifest_text']
      locs += assert_unsigned_manifest coll['unsigned_manifest_text'], coll['uuid']
    end
    assert_operator locs, :>, 0, "no locators found in any manifests"
  end

  test 'index without select returns everything except manifest' do
    authorize_with :active
    get :index
    assert_response :success
    assert json_response['items'].any?
    json_response['items'].each do |coll|
      assert_includes(coll.keys, 'uuid')
      assert_includes(coll.keys, 'name')
      assert_includes(coll.keys, 'created_at')
      refute_includes(coll.keys, 'manifest_text')
    end
  end

  ['', nil, false, 'null'].each do |select|
    test "index with select=#{select.inspect} returns everything except manifest" do
      authorize_with :active
      get :index, params: {select: select}
      assert_response :success
      assert json_response['items'].any?
      json_response['items'].each do |coll|
        assert_includes(coll.keys, 'uuid')
        assert_includes(coll.keys, 'name')
        assert_includes(coll.keys, 'created_at')
        refute_includes(coll.keys, 'manifest_text')
      end
    end
  end

  [["uuid"],
   ["uuid", "manifest_text"],
   '["uuid"]',
   '["uuid", "manifest_text"]'].each do |select|
    test "index with select=#{select.inspect} returns no name" do
      authorize_with :active
      get :index, params: {select: select}
      assert_response :success
      assert json_response['items'].any?
      json_response['items'].each do |coll|
        refute_includes(coll.keys, 'name')
      end
    end
  end

  [0,1,2].each do |limit|
    test "get index with limit=#{limit}" do
      authorize_with :active
      get :index, params: {limit: limit}
      assert_response :success
      assert_equal limit, assigns(:objects).count
      resp = JSON.parse(@response.body)
      assert_equal limit, resp['limit']
    end
  end

  test "items.count == items_available" do
    authorize_with :active
    get :index, params: {limit: 100000}
    assert_response :success
    resp = JSON.parse(@response.body)
    assert_equal resp['items_available'], assigns(:objects).length
    assert_equal resp['items_available'], resp['items'].count
    unique_uuids = resp['items'].collect { |i| i['uuid'] }.compact.uniq
    assert_equal unique_uuids.count, resp['items'].count
  end

  test "items.count == items_available with filters" do
    authorize_with :active
    get :index, params: {
      limit: 100,
      filters: [['uuid','=',collections(:foo_file).uuid]]
    }
    assert_response :success
    assert_equal 1, assigns(:objects).length
    assert_equal 1, json_response['items_available']
    assert_equal 1, json_response['items'].count
  end

  test "get index with limit=2 offset=99999" do
    # Assume there are not that many test fixtures.
    authorize_with :active
    get :index, params: {limit: 2, offset: 99999}
    assert_response :success
    assert_equal 0, assigns(:objects).count
    resp = JSON.parse(@response.body)
    assert_equal 2, resp['limit']
    assert_equal 99999, resp['offset']
  end

  def request_capped_index(params={})
    authorize_with :user1_with_load
    coll1 = collections(:collection_1_of_201)
    Rails.configuration.API.MaxIndexDatabaseRead =
      yield(coll1.manifest_text.size)
    get :index, params: {
      select: %w(uuid manifest_text),
      filters: [["owner_uuid", "=", coll1.owner_uuid]],
      limit: 300,
    }.merge(params)
  end

  test "index with manifest_text limited by max_index_database_read returns non-empty" do
    request_capped_index() { |_| 1 }
    assert_response :success
    assert_equal(1, json_response["items"].size)
    assert_equal(1, json_response["limit"])
    assert_equal(201, json_response["items_available"])
  end

  test "max_index_database_read size check follows same order as real query" do
    authorize_with :user1_with_load
    txt = '.' + ' d41d8cd98f00b204e9800998ecf8427e+0'*1000 + " 0:0:empty.txt\n"
    c = Collection.create! manifest_text: txt, name: '0000000000000000000'
    request_capped_index(select: %w(uuid manifest_text name),
                         order: ['name asc'],
                         filters: [['name','>=',c.name]]) do |_|
      txt.length - 1
    end
    assert_response :success
    assert_equal(1, json_response["items"].size)
    assert_equal(1, json_response["limit"])
    assert_equal(c.uuid, json_response["items"][0]["uuid"])
    # The effectiveness of the test depends on >1 item matching the filters.
    assert_operator(1, :<, json_response["items_available"])
  end

  test "index with manifest_text limited by max_index_database_read" do
    request_capped_index() { |size| (size * 3) + 1 }
    assert_response :success
    assert_equal(3, json_response["items"].size)
    assert_equal(3, json_response["limit"])
    assert_equal(201, json_response["items_available"])
  end

  test "max_index_database_read does not interfere with limit" do
    request_capped_index(limit: 5) { |size| size * 20 }
    assert_response :success
    assert_equal(5, json_response["items"].size)
    assert_equal(5, json_response["limit"])
    assert_equal(201, json_response["items_available"])
  end

  test "max_index_database_read does not interfere with order" do
    request_capped_index(select: %w(uuid manifest_text name),
                         order: "name DESC") { |size| (size * 11) + 1 }
    assert_response :success
    assert_equal(11, json_response["items"].size)
    assert_empty(json_response["items"].reject do |coll|
                   coll["name"] =~ /^Collection_9/
                 end)
    assert_equal(11, json_response["limit"])
    assert_equal(201, json_response["items_available"])
  end

  test "admin can create collection with unsigned manifest" do
    authorize_with :admin
    test_collection = {
      manifest_text: <<-EOS
. d41d8cd98f00b204e9800998ecf8427e+0 0:0:foo.txt
. acbd18db4cc2f85cedef654fccc4a4d8+3 0:3:bar.txt
. acbd18db4cc2f85cedef654fccc4a4d8+3 0:3:bar.txt
./baz acbd18db4cc2f85cedef654fccc4a4d8+3 0:3:bar.txt
EOS
    }
    test_collection[:portable_data_hash] =
      Digest::MD5.hexdigest(test_collection[:manifest_text]) +
      '+' +
      test_collection[:manifest_text].length.to_s

    # post :create will modify test_collection in place, so we save a copy first.
    # Hash.deep_dup is not sufficient as it preserves references of strings (??!?)
    post_collection = Marshal.load(Marshal.dump(test_collection))
    post :create, params: {
      collection: post_collection
    }

    assert_response :success
    assert_nil assigns(:objects)

    response_collection = assigns(:object)

    stored_collection = Collection.select([:uuid, :portable_data_hash, :manifest_text]).
      where(portable_data_hash: response_collection['portable_data_hash']).first

    assert_equal test_collection[:portable_data_hash], stored_collection['portable_data_hash']

    # The manifest in the response will have had permission hints added.
    # Remove any permission hints in the response before comparing it to the source.
    stripped_manifest = stored_collection['manifest_text'].gsub(/\+A[A-Za-z0-9@_-]+/, '')
    assert_equal test_collection[:manifest_text], stripped_manifest

    # TBD: create action should add permission signatures to manifest_text in the response,
    # and we need to check those permission signatures here.
  end

  [:admin, :active].each do |user|
    test "#{user} can get collection using portable data hash" do
      authorize_with user

      foo_collection = collections(:foo_file)

      # Get foo_file using its portable data hash
      get :show, params: {
        id: foo_collection[:portable_data_hash]
      }
      assert_response :success
      assert_not_nil assigns(:object)
      resp = assigns(:object)
      assert_equal foo_collection[:portable_data_hash], resp[:portable_data_hash]
      assert_unsigned_manifest resp[:manifest_text]

      # The manifest in the response will have had permission hints added.
      # Remove any permission hints in the response before comparing it to the source.
      stripped_manifest = resp[:manifest_text].gsub(/\+A[A-Za-z0-9@_-]+/, '')
      assert_equal foo_collection[:manifest_text], stripped_manifest
    end
  end

  test "create with owner_uuid set to owned group" do
    permit_unsigned_manifests
    authorize_with :active
    manifest_text = ". d41d8cd98f00b204e9800998ecf8427e 0:0:foo.txt\n"
    post :create, params: {
      collection: {
        owner_uuid: 'zzzzz-j7d0g-rew6elm53kancon',
        manifest_text: manifest_text,
        portable_data_hash: "d30fe8ae534397864cb96c544f4cf102+47"
      }
    }
    assert_response :success
    resp = JSON.parse(@response.body)
    assert_equal 'zzzzz-j7d0g-rew6elm53kancon', resp['owner_uuid']
  end

  test "create fails with duplicate name" do
    permit_unsigned_manifests
    authorize_with :admin
    manifest_text = ". d41d8cd98f00b204e9800998ecf8427e 0:0:foo.txt\n"
    post :create, params: {
      collection: {
        owner_uuid: 'zzzzz-tpzed-000000000000000',
        manifest_text: manifest_text,
        portable_data_hash: "d30fe8ae534397864cb96c544f4cf102+47",
        name: "foo_file"
      }
    }
    assert_response 422
    response_errors = json_response['errors']
    assert_not_nil response_errors, 'Expected error in response'
    assert(response_errors.first.include?('duplicate key'),
           "Expected 'duplicate key' error in #{response_errors.first}")
  end

  [false, true].each do |unsigned|
    test "create with duplicate name, ensure_unique_name, unsigned=#{unsigned}" do
      permit_unsigned_manifests unsigned
      authorize_with :active
      manifest_text = ". acbd18db4cc2f85cedef654fccc4a4d8+3 0:0:foo.txt\n"
      if !unsigned
        manifest_text = Collection.sign_manifest_only_for_tests manifest_text, api_token(:active)
      end
      post :create, params: {
        collection: {
          owner_uuid: users(:active).uuid,
          manifest_text: manifest_text,
          name: "owned_by_active"
        },
        ensure_unique_name: true
      }
      assert_response :success
      assert_match /^owned_by_active \(\d{4}-\d\d-\d\d.*?Z\)$/, json_response['name']
    end
  end

  test "create with owner_uuid set to group i can_manage" do
    permit_unsigned_manifests
    authorize_with :active
    manifest_text = ". d41d8cd98f00b204e9800998ecf8427e 0:0:foo.txt\n"
    post :create, params: {
      collection: {
        owner_uuid: groups(:active_user_has_can_manage).uuid,
        manifest_text: manifest_text,
        portable_data_hash: "d30fe8ae534397864cb96c544f4cf102+47"
      }
    }
    assert_response :success
    resp = JSON.parse(@response.body)
    assert_equal groups(:active_user_has_can_manage).uuid, resp['owner_uuid']
  end

  test "create with owner_uuid fails on group with only can_read permission" do
    permit_unsigned_manifests
    authorize_with :active
    manifest_text = ". d41d8cd98f00b204e9800998ecf8427e 0:0:foo.txt\n"
    post :create, params: {
      collection: {
        owner_uuid: groups(:all_users).uuid,
        manifest_text: manifest_text,
        portable_data_hash: "d30fe8ae534397864cb96c544f4cf102+47"
      }
    }
    assert_response 403
  end

  test "create with owner_uuid fails on group with no permission" do
    permit_unsigned_manifests
    authorize_with :active
    manifest_text = ". d41d8cd98f00b204e9800998ecf8427e 0:0:foo.txt\n"
    post :create, params: {
      collection: {
        owner_uuid: groups(:public).uuid,
        manifest_text: manifest_text,
        portable_data_hash: "d30fe8ae534397864cb96c544f4cf102+47"
      }
    }
    assert_response 422
  end

  test "admin create with owner_uuid set to group with no permission" do
    permit_unsigned_manifests
    authorize_with :admin
    manifest_text = ". d41d8cd98f00b204e9800998ecf8427e 0:0:foo.txt\n"
    post :create, params: {
      collection: {
        owner_uuid: 'zzzzz-j7d0g-it30l961gq3t0oi',
        manifest_text: manifest_text,
        portable_data_hash: "d30fe8ae534397864cb96c544f4cf102+47"
      }
    }
    assert_response :success
  end

  test "should create with collection passed as json" do
    permit_unsigned_manifests
    authorize_with :active
    post :create, params: {
      collection: <<-EOS
      {
        "manifest_text":". d41d8cd98f00b204e9800998ecf8427e 0:0:foo.txt\n",\
        "portable_data_hash":"d30fe8ae534397864cb96c544f4cf102+47"\
      }
      EOS
    }
    assert_response :success
  end

  test "should fail to create with checksum mismatch" do
    permit_unsigned_manifests
    authorize_with :active
    post :create, params: {
      collection: <<-EOS
      {
        "manifest_text":". d41d8cd98f00b204e9800998ecf8427e 0:0:bar.txt\n",\
        "portable_data_hash":"d30fe8ae534397864cb96c544f4cf102+47"\
      }
      EOS
    }
    assert_response 422
  end

  test "collection UUID is normalized when created" do
    permit_unsigned_manifests
    authorize_with :active
    post :create, params: {
      collection: {
        manifest_text: ". d41d8cd98f00b204e9800998ecf8427e 0:0:foo.txt\n",
        portable_data_hash: "d30fe8ae534397864cb96c544f4cf102+47+Khint+Xhint+Zhint"
      }
    }
    assert_response :success
    assert_not_nil assigns(:object)
    resp = JSON.parse(@response.body)
    assert_equal "d30fe8ae534397864cb96c544f4cf102+47", resp['portable_data_hash']
  end

  test "get full provenance for baz file" do
    authorize_with :active
    get :provenance, params: {id: 'ea10d51bcf88862dbcc36eb292017dfd+45'}
    assert_response :success
    resp = JSON.parse(@response.body)
    assert_not_nil resp['ea10d51bcf88862dbcc36eb292017dfd+45'] # baz
    assert_not_nil resp['fa7aeb5140e2848d39b416daeef4ffc5+45'] # bar
    assert_not_nil resp['1f4b0bc7583c2a7f9102c395f4ffc5e3+45'] # foo
    assert_not_nil resp['zzzzz-8i9sb-cjs4pklxxjykyuq'] # bar->baz
    assert_not_nil resp['zzzzz-8i9sb-aceg2bnq7jt7kon'] # foo->bar
  end

  test "get no provenance for foo file" do
    # spectator user cannot even see baz collection
    authorize_with :spectator
    get :provenance, params: {id: '1f4b0bc7583c2a7f9102c395f4ffc5e3+45'}
    assert_response 404
  end

  test "get partial provenance for baz file" do
    # spectator user can see bar->baz job, but not foo->bar job
    authorize_with :spectator
    get :provenance, params: {id: 'ea10d51bcf88862dbcc36eb292017dfd+45'}
    assert_response :success
    resp = JSON.parse(@response.body)
    assert_not_nil resp['ea10d51bcf88862dbcc36eb292017dfd+45'] # baz
    assert_not_nil resp['fa7aeb5140e2848d39b416daeef4ffc5+45'] # bar
    assert_not_nil resp['zzzzz-8i9sb-cjs4pklxxjykyuq']     # bar->baz
    assert_nil resp['zzzzz-8i9sb-aceg2bnq7jt7kon']         # foo->bar
    assert_nil resp['1f4b0bc7583c2a7f9102c395f4ffc5e3+45'] # foo
  end

  test "search collections with 'any' operator" do
    expect_pdh = collections(:docker_image).portable_data_hash
    authorize_with :active
    get :index, params: {
      where: { any: ['contains', expect_pdh[5..25]] }
    }
    assert_response :success
    found = assigns(:objects)
    assert_equal 1, found.count
    assert_equal expect_pdh, found.first.portable_data_hash
  end

  [false, true].each do |permit_unsigned|
    test "create collection with signed manifest, permit_unsigned=#{permit_unsigned}" do
      permit_unsigned_manifests permit_unsigned
      authorize_with :active
      locators = %w(
      d41d8cd98f00b204e9800998ecf8427e+0
      acbd18db4cc2f85cedef654fccc4a4d8+3
      ea10d51bcf88862dbcc36eb292017dfd+45)

      unsigned_manifest = locators.map { |loc|
        ". " + loc + " 0:0:foo.txt\n"
      }.join()
      manifest_uuid = Digest::MD5.hexdigest(unsigned_manifest) +
        '+' +
        unsigned_manifest.length.to_s

      # Build a manifest with both signed and unsigned locators.
      signing_opts = {
        key: Rails.configuration.Collections.BlobSigningKey,
        api_token: api_token(:active),
      }
      signed_locators = locators.collect do |x|
        Blob.sign_locator x, signing_opts
      end
      if permit_unsigned
        # Leave a non-empty blob unsigned.
        signed_locators[1] = locators[1]
      else
        # Leave the empty blob unsigned. This should still be allowed.
        signed_locators[0] = locators[0]
      end
      signed_manifest =
        ". " + signed_locators[0] + " 0:0:foo.txt\n" +
        ". " + signed_locators[1] + " 0:0:foo.txt\n" +
        ". " + signed_locators[2] + " 0:0:foo.txt\n"

      post :create, params: {
        collection: {
          manifest_text: signed_manifest,
          portable_data_hash: manifest_uuid,
        }
      }
      assert_response :success
      assert_not_nil assigns(:object)
      resp = JSON.parse(@response.body)
      assert_equal manifest_uuid, resp['portable_data_hash']
      # All of the signatures in the output must be valid.
      resp['manifest_text'].lines.each do |entry|
        m = /([[:xdigit:]]{32}\+\S+)/.match(entry)
        if m && m[0].index('+A')
          assert Blob.verify_signature m[0], signing_opts
        end
      end
    end
  end

  test "create collection with signed manifest and explicit TTL" do
    authorize_with :active
    locators = %w(
      d41d8cd98f00b204e9800998ecf8427e+0
      acbd18db4cc2f85cedef654fccc4a4d8+3
      ea10d51bcf88862dbcc36eb292017dfd+45)

    unsigned_manifest = locators.map { |loc|
      ". " + loc + " 0:0:foo.txt\n"
    }.join()
    manifest_uuid = Digest::MD5.hexdigest(unsigned_manifest) +
      '+' +
      unsigned_manifest.length.to_s

    # build a manifest with both signed and unsigned locators.
    # TODO(twp): in phase 4, all locators will need to be signed, so
    # this test should break and will need to be rewritten. Issue #2755.
    signing_opts = {
      key: Rails.configuration.Collections.BlobSigningKey,
      api_token: api_token(:active),
      ttl: 3600   # 1 hour
    }
    signed_manifest =
      ". " + locators[0] + " 0:0:foo.txt\n" +
      ". " + Blob.sign_locator(locators[1], signing_opts) + " 0:0:foo.txt\n" +
      ". " + Blob.sign_locator(locators[2], signing_opts) + " 0:0:foo.txt\n"

    post :create, params: {
      collection: {
        manifest_text: signed_manifest,
        portable_data_hash: manifest_uuid,
      }
    }
    assert_response :success
    assert_not_nil assigns(:object)
    resp = JSON.parse(@response.body)
    assert_equal manifest_uuid, resp['portable_data_hash']
    # All of the signatures in the output must be valid.
    resp['manifest_text'].lines.each do |entry|
      m = /([[:xdigit:]]{32}\+\S+)/.match(entry)
      if m && m[0].index('+A')
        assert Blob.verify_signature m[0], signing_opts
      end
    end
  end

  test "create fails with invalid signature" do
    authorize_with :active
    signing_opts = {
      key: Rails.configuration.Collections.BlobSigningKey,
      api_token: api_token(:active),
    }

    # Generate a locator with a bad signature.
    unsigned_locator = "acbd18db4cc2f85cedef654fccc4a4d8+3"
    bad_locator = unsigned_locator + "+Affffffffffffffffffffffffffffffffffffffff@ffffffff"
    assert !Blob.verify_signature(bad_locator, signing_opts)

    # Creating a collection with this locator should
    # produce 403 Permission denied.
    unsigned_manifest = ". #{unsigned_locator} 0:0:foo.txt\n"
    manifest_uuid = Digest::MD5.hexdigest(unsigned_manifest) +
      '+' +
      unsigned_manifest.length.to_s

    bad_manifest = ". #{bad_locator} 0:0:foo.txt\n"
    post :create, params: {
      collection: {
        manifest_text: bad_manifest,
        portable_data_hash: manifest_uuid
      }
    }

    assert_response 403
  end

  test "create fails with uuid of signed manifest" do
    authorize_with :active
    signing_opts = {
      key: Rails.configuration.Collections.BlobSigningKey,
      api_token: api_token(:active),
    }

    unsigned_locator = "d41d8cd98f00b204e9800998ecf8427e+0"
    signed_locator = Blob.sign_locator(unsigned_locator, signing_opts)
    signed_manifest = ". #{signed_locator} 0:0:foo.txt\n"
    manifest_uuid = Digest::MD5.hexdigest(signed_manifest) +
      '+' +
      signed_manifest.length.to_s

    post :create, params: {
      collection: {
        manifest_text: signed_manifest,
        portable_data_hash: manifest_uuid
      }
    }

    assert_response 422
  end

  test "reject manifest with unsigned block as stream name" do
    authorize_with :active
    post :create, params: {
      collection: {
        manifest_text: "00000000000000000000000000000000+1234 d41d8cd98f00b204e9800998ecf8427e+0 0:0:foo.txt\n"
      }
    }
    assert_includes [422, 403], response.code.to_i
  end

  test "multiple locators per line" do
    permit_unsigned_manifests
    authorize_with :active
    locators = %w(
      d41d8cd98f00b204e9800998ecf8427e+0
      acbd18db4cc2f85cedef654fccc4a4d8+3
      ea10d51bcf88862dbcc36eb292017dfd+45)

    manifest_text = [".", *locators, "0:0:foo.txt\n"].join(" ")
    manifest_uuid = Digest::MD5.hexdigest(manifest_text) +
      '+' +
      manifest_text.length.to_s

    test_collection = {
      manifest_text: manifest_text,
      portable_data_hash: manifest_uuid,
    }
    post_collection = Marshal.load(Marshal.dump(test_collection))
    post :create, params: {
      collection: post_collection
    }
    assert_response :success
    assert_not_nil assigns(:object)
    resp = JSON.parse(@response.body)
    assert_equal manifest_uuid, resp['portable_data_hash']

    # The manifest in the response will have had permission hints added.
    # Remove any permission hints in the response before comparing it to the source.
    stripped_manifest = resp['manifest_text'].gsub(/\+A[A-Za-z0-9@_-]+/, '')
    assert_equal manifest_text, stripped_manifest
  end

  test 'Reject manifest with unsigned blob' do
    permit_unsigned_manifests false
    authorize_with :active
    unsigned_manifest = ". 0cc175b9c0f1b6a831c399e269772661+1 0:1:a.txt\n"
    manifest_uuid = Digest::MD5.hexdigest(unsigned_manifest)
    post :create, params: {
      collection: {
        manifest_text: unsigned_manifest,
        portable_data_hash: manifest_uuid,
      }
    }
    assert_response 403,
    "Creating a collection with unsigned blobs should respond 403"
    assert_empty Collection.where('uuid like ?', manifest_uuid+'%'),
    "Collection should not exist in database after failed create"
  end

  test 'List expired collection returns empty list' do
    authorize_with :active
    get :index, params: {
      where: {name: 'expired_collection'},
    }
    assert_response :success
    found = assigns(:objects)
    assert_equal 0, found.count
  end

  test 'Show expired collection returns 404' do
    authorize_with :active
    get :show, params: {
      id: 'zzzzz-4zz18-mto52zx1s7sn3ih',
    }
    assert_response 404
  end

  test 'Update expired collection returns 404' do
    authorize_with :active
    post :update, params: {
      id: 'zzzzz-4zz18-mto52zx1s7sn3ih',
      collection: {
        name: "still expired"
      }
    }
    assert_response 404
  end

  test 'List collection with future expiration time succeeds' do
    authorize_with :active
    get :index, params: {
      where: {name: 'collection_expires_in_future'},
    }
    found = assigns(:objects)
    assert_equal 1, found.count
  end


  test 'Show collection with future expiration time succeeds' do
    authorize_with :active
    get :show, params: {
      id: 'zzzzz-4zz18-padkqo7yb8d9i3j',
    }
    assert_response :success
  end

  test 'Update collection with future expiration time succeeds' do
    authorize_with :active
    post :update, params: {
      id: 'zzzzz-4zz18-padkqo7yb8d9i3j',
      collection: {
        name: "still not expired"
      }
    }
    assert_response :success
  end

  test "get collection and verify that file_names is not included" do
    authorize_with :active
    get :show, params: {id: collections(:foo_file).uuid}
    assert_response :success
    assert_equal collections(:foo_file).uuid, json_response['uuid']
    assert_nil json_response['file_names']
    assert json_response['manifest_text']
  end

  [
    [2**8, :success],
    [2**18, 422],
  ].each do |description_size, expected_response|
    # Descriptions are not part of search indexes. Skip until
    # full-text search is implemented, at which point replace with a
    # search in description.
    skip "create collection with description size #{description_size}
          and expect response #{expected_response}" do
      authorize_with :active

      description = 'here is a collection with a very large description'
      while description.length < description_size
        description = description + description
      end

      post :create, params: {
        collection: {
          manifest_text: ". d41d8cd98f00b204e9800998ecf8427e+0 0:0:foo.txt\n",
          description: description,
        }
      }

      assert_response expected_response
    end
  end

  [1, 5, nil].each do |ask|
    test "Set replication_desired=#{ask.inspect}" do
      Rails.configuration.Collections.DefaultReplication = 2
      authorize_with :active
      put :update, params: {
        id: collections(:replication_undesired_unconfirmed).uuid,
        collection: {
          replication_desired: ask,
        },
      }
      assert_response :success
      assert_equal ask, json_response['replication_desired']
    end
  end

  test "get collection with properties" do
    authorize_with :active
    get :show, params: {id: collections(:collection_with_one_property).uuid}
    assert_response :success
    assert_not_nil json_response['uuid']
    assert_equal 'value1', json_response['properties']['property1']
  end

  [
    {'property_1' => 'value_1'},
    "{\"property_1\":\"value_1\"}",
  ].each do |p|
    test "create collection with valid properties param #{p.inspect}" do
      authorize_with :active
      manifest_text = ". d41d8cd98f00b204e9800998ecf8427e 0:0:foo.txt\n"
      post :create, params: {
        collection: {
          manifest_text: manifest_text,
          portable_data_hash: "d30fe8ae534397864cb96c544f4cf102+47",
          properties: p
        }
      }
      assert_response :success
      assert_not_nil json_response['uuid']
      assert_equal Hash, json_response['properties'].class, 'Collection properties attribute should be of type hash'
      assert_equal 'value_1', json_response['properties']['property_1']
    end
  end

  [
    false,
    [],
    42,
    'some string',
    '["json", "encoded", "array"]',
  ].each do |p|
    test "create collection with non-valid properties param #{p.inspect}" do
      authorize_with :active
      post :create, params: {
        collection: {
          name: "test collection with non-valid properties param '#{p.inspect}'",
          manifest_text: '',
          properties: p
        }
      }
      assert_response 422
      response_errors = json_response['errors']
      assert_not_nil response_errors, 'Expected error in response'
    end
  end

  [
    [". d41d8cd98f00b204e9800998ecf8427e 0:34:foo.txt\n", 1, 34],
    [". d41d8cd98f00b204e9800998ecf8427e 0:34:foo.txt 0:30:foo.txt 0:30:foo1.txt 0:30:foo2.txt 0:30:foo3.txt 0:30:foo4.txt\n", 5, 184],
    [". d41d8cd98f00b204e9800998ecf8427e 0:0:.\n", 0, 0]
  ].each do |manifest, count, size|
    test "create collection with valid manifest #{manifest} and expect file stats" do
      authorize_with :active
      post :create, params: {
        collection: {
          manifest_text: manifest
        }
      }
      assert_response 200
      assert_equal count, json_response['file_count']
      assert_equal size, json_response['file_size_total']
    end
  end

  test "update collection manifest and expect new file stats" do
    authorize_with :active
    post :update, params: {
      id: collections(:collection_owned_by_active_with_file_stats).uuid,
      collection: {
        manifest_text: ". d41d8cd98f00b204e9800998ecf8427e 0:34:foo.txt\n"
      }
    }
    assert_response 200
    assert_equal 1, json_response['file_count']
    assert_equal 34, json_response['file_size_total']
  end

  [
    ['file_count', 1],
    ['file_size_total', 34]
  ].each do |attribute, val|
    test "create collection with #{attribute} and expect overwrite" do
      authorize_with :active
      post :create, params: {
        collection: {
          manifest_text: ". d41d8cd98f00b204e9800998ecf8427e 0:34:foo.txt\n",
          "#{attribute}": 10
        }
      }
      assert_response 200
      assert_equal val, json_response[attribute]
    end
  end

  [
    ['file_count', 1],
    ['file_size_total', 3]
  ].each do |attribute, val|
    test "update collection with #{attribute} and expect ignore" do
      authorize_with :active
      post :update, params: {
        id: collections(:collection_owned_by_active_with_file_stats).uuid,
        collection: {
          "#{attribute}": 10
        }
      }
      assert_response 200
      assert_equal val, json_response[attribute]
    end
  end

  [
    ['file_count', 1],
    ['file_size_total', 34]
  ].each do |attribute, val|
    test "update collection with #{attribute} and manifest and expect manifest values" do
      authorize_with :active
      post :update, params: {
        id: collections(:collection_owned_by_active_with_file_stats).uuid,
        collection: {
          manifest_text: ". d41d8cd98f00b204e9800998ecf8427e 0:34:foo.txt\n",
          "#{attribute}": 10
        }
      }
      assert_response 200
      assert_equal val, json_response[attribute]
    end
  end

  [
    ". 0:0:foo.txt",
    ". d41d8cd98f00b204e9800998ecf8427e foo.txt",
    "d41d8cd98f00b204e9800998ecf8427e 0:0:foo.txt",
    ". d41d8cd98f00b204e9800998ecf8427e 0:0:foo.txt",
  ].each do |manifest_text|
    test "create collection with invalid manifest #{manifest_text} and expect error" do
      authorize_with :active
      post :create, params: {
        collection: {
          manifest_text: manifest_text,
          portable_data_hash: "d41d8cd98f00b204e9800998ecf8427e+0"
        }
      }
      assert_response 422
      response_errors = json_response['errors']
      assert_not_nil response_errors, 'Expected error in response'
      assert(response_errors.first.include?('Invalid manifest'),
             "Expected 'Invalid manifest' error in #{response_errors.first}")
    end
  end

  [
    [nil, "d41d8cd98f00b204e9800998ecf8427e+0"],
    ["", "d41d8cd98f00b204e9800998ecf8427e+0"],
    [". d41d8cd98f00b204e9800998ecf8427e 0:0:foo.txt\n", "d30fe8ae534397864cb96c544f4cf102+47"],
  ].each do |manifest_text, pdh|
    test "create collection with valid manifest #{manifest_text.inspect} and expect success" do
      authorize_with :active
      post :create, params: {
        collection: {
          manifest_text: manifest_text,
          portable_data_hash: pdh
        }
      }
      assert_response 200
    end
  end

  [
    ". 0:0:foo.txt",
    ". d41d8cd98f00b204e9800998ecf8427e foo.txt",
    "d41d8cd98f00b204e9800998ecf8427e 0:0:foo.txt",
    ". d41d8cd98f00b204e9800998ecf8427e 0:0:foo.txt",
  ].each do |manifest_text|
    test "update collection with invalid manifest #{manifest_text} and expect error" do
      authorize_with :active
      post :update, params: {
        id: 'zzzzz-4zz18-bv31uwvy3neko21',
        collection: {
          manifest_text: manifest_text,
        }
      }
      assert_response 422
      response_errors = json_response['errors']
      assert_not_nil response_errors, 'Expected error in response'
      assert(response_errors.first.include?('Invalid manifest'),
             "Expected 'Invalid manifest' error in #{response_errors.first}")
    end
  end

  [
    nil,
    "",
    ". d41d8cd98f00b204e9800998ecf8427e 0:0:foo.txt\n",
  ].each do |manifest_text|
    test "update collection with valid manifest #{manifest_text.inspect} and expect success" do
      authorize_with :active
      post :update, params: {
        id: 'zzzzz-4zz18-bv31uwvy3neko21',
        collection: {
          manifest_text: manifest_text,
        }
      }
      assert_response 200
    end
  end

  [true, false].each do |include_trash|
    test "get trashed collection with include_trash=#{include_trash}" do
      uuid = 'zzzzz-4zz18-mto52zx1s7sn3ih' # expired_collection
      authorize_with :active
      get :show, params: {
        id: uuid,
        include_trash: include_trash,
      }
      if include_trash
        assert_response 200
      else
        assert_response 404
      end
    end
  end

  [:admin, :active].each do |user|
    test "get trashed collection via filters and #{user} user without including its past versions" do
      uuid = 'zzzzz-4zz18-mto52zx1s7sn3ih' # expired_collection
      authorize_with user
      get :index, params: {
        filters: [["current_version_uuid", "=", uuid]],
        include_trash: true,
      }
      assert_response 200
      # Only the current version is returned
      assert_equal 1, json_response["items"].size
    end
  end

  [:admin, :active].each do |user|
    test "get trashed collection via filters and #{user} user, including its past versions" do
      uuid = 'zzzzz-4zz18-mto52zx1s7sn3ih' # expired_collection
      authorize_with :admin
      get :index, params: {
        filters: [["current_version_uuid", "=", uuid]],
        include_trash: true,
        include_old_versions: true,
      }
      assert_response 200
      # Both current & past version are returned
      assert_equal 2, json_response["items"].size
    end
  end

  test "trash collection also trash its past versions" do
    uuid = collections(:collection_owned_by_active).uuid
    authorize_with :active
    versions = Collection.where(current_version_uuid: uuid)
    assert_equal 2, versions.size
    versions.each do |col|
      refute col.is_trashed
    end
    post :trash, params: {
      id: uuid,
    }
    assert_response 200
    versions = Collection.where(current_version_uuid: uuid)
    assert_equal 2, versions.size
    versions.each do |col|
      assert col.is_trashed
    end
  end

  test 'get trashed collection without include_trash' do
    uuid = 'zzzzz-4zz18-mto52zx1s7sn3ih' # expired_collection
    authorize_with :active
    get :show, params: {
      id: uuid,
    }
    assert_response 404
  end

  test 'trash collection using http DELETE verb' do
    uuid = collections(:collection_owned_by_active).uuid
    authorize_with :active
    delete :destroy, params: {
      id: uuid,
    }
    assert_response 200
    c = Collection.find_by_uuid(uuid)
    assert_operator c.trash_at, :<, db_current_time
    assert_equal c.delete_at, c.trash_at + Rails.configuration.Collections.BlobSigningTTL
  end

  test 'delete long-trashed collection immediately using http DELETE verb' do
    uuid = 'zzzzz-4zz18-mto52zx1s7sn3ih' # expired_collection
    authorize_with :active
    delete :destroy, params: {
      id: uuid,
    }
    assert_response 200
    c = Collection.find_by_uuid(uuid)
    assert_operator c.trash_at, :<, db_current_time
    assert_operator c.delete_at, :<, db_current_time
  end

  ['zzzzz-4zz18-mto52zx1s7sn3ih', # expired_collection
   :empty_collection_name_in_active_user_home_project,
  ].each do |fixture|
    test "trash collection #{fixture} via trash action with grace period" do
      if fixture.is_a? String
        uuid = fixture
      else
        uuid = collections(fixture).uuid
      end
      authorize_with :active
      time_before_trashing = db_current_time
      post :trash, params: {
        id: uuid,
      }
      assert_response 200
      c = Collection.find_by_uuid(uuid)
      assert_operator c.trash_at, :<, db_current_time
      assert_operator c.delete_at, :>=, time_before_trashing + Rails.configuration.Collections.DefaultTrashLifetime
    end
  end

  test 'untrash a trashed collection' do
    authorize_with :active
    post :untrash, params: {
      id: collections(:expired_collection).uuid,
    }
    assert_response 200
    assert_equal false, json_response['is_trashed']
    assert_nil json_response['trash_at']
  end

  test 'untrash error on not trashed collection' do
    authorize_with :active
    post :untrash, params: {
      id: collections(:collection_owned_by_active).uuid,
    }
    assert_response 422
  end

  [:active, :admin].each do |user|
    test "get trashed collections as #{user}" do
      authorize_with user
      get :index, params: {
        filters: [["is_trashed", "=", true]],
        include_trash: true,
      }
      assert_response :success

      items = []
      json_response["items"].each do |coll|
        items << coll['uuid']
      end

      assert_includes(items, collections('unique_expired_collection')['uuid'])
      if user == :admin
        assert_includes(items, collections('unique_expired_collection2')['uuid'])
      else
        assert_not_includes(items, collections('unique_expired_collection2')['uuid'])
      end
    end
  end

  test 'untrash collection with same name as another with no ensure unique name' do
    authorize_with :active
    post :untrash, params: {
      id: collections(:trashed_collection_to_test_name_conflict_on_untrash).uuid,
    }
    assert_response 422
  end

  test 'untrash collection with same name as another with ensure unique name' do
    authorize_with :active
    post :untrash, params: {
      id: collections(:trashed_collection_to_test_name_conflict_on_untrash).uuid,
      ensure_unique_name: true
    }
    assert_response 200
    assert_equal false, json_response['is_trashed']
    assert_nil json_response['trash_at']
    assert_nil json_response['delete_at']
    assert_match /^same name for trashed and persisted collections \(\d{4}-\d\d-\d\d.*?Z\)$/, json_response['name']
  end

  test 'cannot show collection in trashed subproject' do
    authorize_with :active
    get :show, params: {
      id: collections(:collection_in_trashed_subproject).uuid,
      format: :json
    }
    assert_response 404
  end

  test 'can show collection in untrashed subproject' do
    authorize_with :active
    Group.find_by_uuid(groups(:trashed_project).uuid).update! is_trashed: false
    get :show, params: {
      id: collections(:collection_in_trashed_subproject).uuid,
      format: :json,
    }
    assert_response :success
  end

  test 'cannot index collection in trashed subproject' do
    authorize_with :active
    get :index, params: { limit: 1000 }
    assert_response :success
    item_uuids = json_response['items'].map do |item|
      item['uuid']
    end
    assert_not_includes(item_uuids, collections(:collection_in_trashed_subproject).uuid)
  end

  test 'can index collection in untrashed subproject' do
    authorize_with :active
    Group.find_by_uuid(groups(:trashed_project).uuid).update! is_trashed: false
    get :index, params: { limit: 1000 }
    assert_response :success
    item_uuids = json_response['items'].map do |item|
      item['uuid']
    end
    assert_includes(item_uuids, collections(:collection_in_trashed_subproject).uuid)
  end

  test 'can index trashed subproject collection with include_trash' do
    authorize_with :active
    get :index, params: {
          include_trash: true,
          limit: 1000
        }
    assert_response :success
    item_uuids = json_response['items'].map do |item|
      item['uuid']
    end
    assert_includes(item_uuids, collections(:collection_in_trashed_subproject).uuid)
  end

  test 'can get collection with past versions' do
    authorize_with :active
    get :index, params: {
      filters: [['current_version_uuid','=',collections(:collection_owned_by_active).uuid]],
      include_old_versions: true
    }
    assert_response :success
    assert_equal 2, assigns(:objects).length
    assert_equal 2, json_response['items_available']
    assert_equal 2, json_response['items'].count
    json_response['items'].each do |c|
      assert_equal collections(:collection_owned_by_active).uuid,
                   c['current_version_uuid'],
                   'response includes a version from a different collection'
    end
  end

  test 'can get old version collection by uuid' do
    authorize_with :active
    get :show, params: {
      id: collections(:collection_owned_by_active_past_version_1).uuid,
    }
    assert_response :success
    assert_equal collections(:collection_owned_by_active_past_version_1).name,
                  json_response['name']
  end

  test 'can get old version collection by PDH' do
    authorize_with :active
    get :show, params: {
      id: collections(:collection_owned_by_active_past_version_1).portable_data_hash,
    }
    assert_response :success
    assert_equal collections(:collection_owned_by_active_past_version_1).portable_data_hash,
                  json_response['portable_data_hash']
  end

  test 'version and current_version_uuid are ignored at creation time' do
    permit_unsigned_manifests
    authorize_with :active
    manifest_text = ". d41d8cd98f00b204e9800998ecf8427e 0:0:foo.txt\n"
    post :create, params: {
      collection: {
        name: 'Test collection',
        version: 42,
        current_version_uuid: collections(:collection_owned_by_active).uuid,
        manifest_text: manifest_text,
      }
    }
    assert_response :success
    resp = JSON.parse(@response.body)
    assert_equal 1, resp['version']
    assert_equal resp['uuid'], resp['current_version_uuid']
  end

  test "update collection with versioning enabled" do
    Rails.configuration.Collections.CollectionVersioning = true
    Rails.configuration.Collections.PreserveVersionIfIdle = 1 # 1 second

    col = collections(:collection_owned_by_active)
    assert_equal 2, col.version
    assert col.modified_at < Time.now - 1.second

    token = api_client_authorizations(:active).v2token
    signed = Blob.sign_locator(
      'acbd18db4cc2f85cedef654fccc4a4d8+3',
      key: Rails.configuration.Collections.BlobSigningKey,
      api_token: token)
    authorize_with_token token
    put :update, params: {
          id: col.uuid,
          collection: {
            manifest_text: ". #{signed} 0:3:foo.txt\n",
          },
        }
    assert_response :success
    assert_equal 3, json_response['version']
  end

  test "delete collection with versioning enabled" do
    Rails.configuration.Collections.CollectionVersioning = true
    Rails.configuration.Collections.PreserveVersionIfIdle = 1 # 1 second

    col = collections(:collection_owned_by_active)
    assert_equal 2, col.version
    assert col.modified_at < Time.now - 1.second

    authorize_with(:active)
    post :trash, params: {
      id: col.uuid,
    }
    assert_response :success
    assert_equal col.version, json_response['version'], 'Trashing a collection should not create a new version'
  end

  [['<', :<],
   ['<=', :<=],
   ['>', :>],
   ['>=', :>=],
   ['=', :==]].each do |op, rubyop|
    test "filter collections by replication_desired #{op} replication_confirmed" do
      authorize_with(:active)
      get :index, params: {
            filters: [["(replication_desired #{op} replication_confirmed)", "=", true]],
          }
      assert_response :success
      json_response["items"].each do |c|
        assert_operator(c["replication_desired"], rubyop, c["replication_confirmed"])
      end
    end
  end

  ["(replication_desired < bogus)",
   "replication_desired < replication_confirmed",
   "(replication_desired < replication_confirmed",
   "(replication_desired ! replication_confirmed)",
   "(replication_desired <)",
   "(replication_desired < manifest_text)",
   "(manifest_text < manifest_text)", # currently only numeric attrs are supported
   "(replication_desired < 2)", # currently only attrs are supported, not literals
   "(1 < 2)",
  ].each do |expr|
    test "invalid filter expression #{expr}" do
      authorize_with(:active)
      get :index, params: {
            filters: [[expr, "=", true]],
          }
      assert_response 422
    end
  end

  test "invalid op/arg with filter expression" do
    authorize_with(:active)
    get :index, params: {
          filters: [["replication_desired < replication_confirmed", "!=", false]],
        }
    assert_response 422
  end

  ["storage_classes_desired", "storage_classes_confirmed"].each do |attr|
    test "filter collections by #{attr}" do
      authorize_with(:active)
      get :index, params: {
            filters: [[attr, "=", '["default"]']]
          }
      assert_response :success
      assert_not_equal 0, json_response["items"].length
      json_response["items"].each do |c|
        assert_equal ["default"], c[attr]
      end
    end
  end

  test "select param is respected in 'show' response" do
    authorize_with :active
    get :show, params: {
          id: collections(:collection_owned_by_active).uuid,
          select: ["name"],
        }
    assert_response :success
    assert_raises ActiveModel::MissingAttributeError do
      assigns(:object).manifest_text
    end
    assert_nil json_response["manifest_text"]
    assert_nil json_response["properties"]
    assert_equal collections(:collection_owned_by_active).name, json_response["name"]
  end

  test "select param is respected in 'update' response" do
    authorize_with :active
    post :update, params: {
          id: collections(:collection_owned_by_active).uuid,
          collection: {
            manifest_text: ". d41d8cd98f00b204e9800998ecf8427e+0 0:0:foobar.txt\n",
          },
          select: ["name"],
        }
    assert_response :success
    assert_nil json_response["manifest_text"]
    assert_nil json_response["properties"]
    assert_equal collections(:collection_owned_by_active).name, json_response["name"]
  end

  [nil,
   [],
   ["is_trashed", "trash_at"],
   ["is_trashed", "trash_at", "portable_data_hash"],
   ["portable_data_hash"],
   ["portable_data_hash", "manifest_text"],
  ].each do |select|
    test "select=#{select.inspect} param is respected in 'get by pdh' response" do
      authorize_with :active
      get :show, params: {
            id: collections(:collection_owned_by_active).portable_data_hash,
            select: select,
          }
      assert_response :success
      if !select || select.index("manifest_text")
        assert_not_nil json_response["manifest_text"]
      else
        assert_nil json_response["manifest_text"]
      end
    end
  end
end
