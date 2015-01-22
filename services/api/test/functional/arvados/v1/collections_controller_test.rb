require 'test_helper'

class Arvados::V1::CollectionsControllerTest < ActionController::TestCase

  def permit_unsigned_manifests isok=true
    # Set security model for the life of a test.
    Rails.configuration.permit_create_collection_with_unsigned_manifest = isok
  end

  def assert_signed_manifest manifest_text, label=''
    assert_not_nil manifest_text, "#{label} manifest_text was nil"
    manifest_text.scan(/ [[:xdigit:]]{32}\S*/) do |tok|
      assert_match(/\+A[[:xdigit:]]+@[[:xdigit:]]{8}\b/, tok,
                   "Locator in #{label} manifest_text was not signed")
    end
  end

  test "should get index" do
    authorize_with :active
    get :index
    assert_response :success
    assert(assigns(:objects).andand.any?, "no Collections returned in index")
    refute(json_response["items"].any? { |c| c.has_key?("manifest_text") },
           "basic Collections index included manifest_text")
  end

  test "collections.get returns signed locators" do
    permit_unsigned_manifests
    authorize_with :active
    get :show, {id: collections(:foo_file).uuid}
    assert_response :success
    assert_signed_manifest json_response['manifest_text'], 'foo_file'
  end

  test "index with manifest_text selected returns signed locators" do
    columns = %w(uuid owner_uuid manifest_text)
    authorize_with :active
    get :index, select: columns
    assert_response :success
    assert(assigns(:objects).andand.any?,
           "no Collections returned for index with columns selected")
    json_response["items"].each do |coll|
      assert_equal(columns, columns & coll.keys,
                   "Collections index did not respect selected columns")
      assert_signed_manifest coll['manifest_text'], coll['uuid']
    end
  end

  [0,1,2].each do |limit|
    test "get index with limit=#{limit}" do
      authorize_with :active
      get :index, limit: limit
      assert_response :success
      assert_equal limit, assigns(:objects).count
      resp = JSON.parse(@response.body)
      assert_equal limit, resp['limit']
    end
  end

  test "items.count == items_available" do
    authorize_with :active
    get :index, limit: 100000
    assert_response :success
    resp = JSON.parse(@response.body)
    assert_equal resp['items_available'], assigns(:objects).length
    assert_equal resp['items_available'], resp['items'].count
    unique_uuids = resp['items'].collect { |i| i['uuid'] }.compact.uniq
    assert_equal unique_uuids.count, resp['items'].count
  end

  test "items.count == items_available with filters" do
    authorize_with :active
    get :index, {
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
    get :index, limit: 2, offset: 99999
    assert_response :success
    assert_equal 0, assigns(:objects).count
    resp = JSON.parse(@response.body)
    assert_equal 2, resp['limit']
    assert_equal 99999, resp['offset']
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
    post :create, {
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
      get :show, {
        id: foo_collection[:portable_data_hash]
      }
      assert_response :success
      assert_not_nil assigns(:object)
      resp = assigns(:object)
      assert_equal foo_collection[:portable_data_hash], resp['portable_data_hash']
      assert_signed_manifest resp['manifest_text']

      # The manifest in the response will have had permission hints added.
      # Remove any permission hints in the response before comparing it to the source.
      stripped_manifest = resp['manifest_text'].gsub(/\+A[A-Za-z0-9@_-]+/, '')
      assert_equal foo_collection[:manifest_text], stripped_manifest
    end
  end

  test "create with owner_uuid set to owned group" do
    permit_unsigned_manifests
    authorize_with :active
    manifest_text = ". d41d8cd98f00b204e9800998ecf8427e 0:0:foo.txt\n"
    post :create, {
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
    post :create, {
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
        manifest_text = Collection.sign_manifest manifest_text, api_token(:active)
      end
      post :create, {
        collection: {
          owner_uuid: users(:active).uuid,
          manifest_text: manifest_text,
          name: "owned_by_active"
        },
        ensure_unique_name: true
      }
      assert_response :success
      assert_equal 'owned_by_active (2)', json_response['name']
    end
  end

  test "create with owner_uuid set to group i can_manage" do
    permit_unsigned_manifests
    authorize_with :active
    manifest_text = ". d41d8cd98f00b204e9800998ecf8427e 0:0:foo.txt\n"
    post :create, {
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
    post :create, {
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
    post :create, {
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
    post :create, {
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
    post :create, {
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
    post :create, {
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
    post :create, {
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
    get :provenance, id: 'ea10d51bcf88862dbcc36eb292017dfd+45'
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
    get :provenance, id: '1f4b0bc7583c2a7f9102c395f4ffc5e3+45'
    assert_response 404
  end

  test "get partial provenance for baz file" do
    # spectator user can see bar->baz job, but not foo->bar job
    authorize_with :spectator
    get :provenance, id: 'ea10d51bcf88862dbcc36eb292017dfd+45'
    assert_response :success
    resp = JSON.parse(@response.body)
    assert_not_nil resp['ea10d51bcf88862dbcc36eb292017dfd+45'] # baz
    assert_not_nil resp['fa7aeb5140e2848d39b416daeef4ffc5+45'] # bar
    assert_not_nil resp['zzzzz-8i9sb-cjs4pklxxjykyuq']     # bar->baz
    assert_nil resp['zzzzz-8i9sb-aceg2bnq7jt7kon']         # foo->bar
    assert_nil resp['1f4b0bc7583c2a7f9102c395f4ffc5e3+45'] # foo
  end

  test "search collections with 'any' operator" do
    authorize_with :active
    get :index, {
      where: { any: ['contains', 'd0bc8c7f34be170a7b7b'] }
    }
    assert_response :success
    found = assigns(:objects).collect(&:portable_data_hash)
    assert_equal 1, found.count
    assert_equal true, !!found.index('5bd9c1ad0bc8c7f34be170a7b7b39089+45')
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
        key: Rails.configuration.blob_signing_key,
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

      post :create, {
        collection: {
          manifest_text: signed_manifest,
          portable_data_hash: manifest_uuid,
        }
      }
      assert_response :success
      assert_not_nil assigns(:object)
      resp = JSON.parse(@response.body)
      assert_equal manifest_uuid, resp['portable_data_hash']
      # All of the locators in the output must be signed.
      resp['manifest_text'].lines.each do |entry|
        m = /([[:xdigit:]]{32}\+\S+)/.match(entry)
        if m
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
      key: Rails.configuration.blob_signing_key,
      api_token: api_token(:active),
      ttl: 3600   # 1 hour
    }
    signed_manifest =
      ". " + locators[0] + " 0:0:foo.txt\n" +
      ". " + Blob.sign_locator(locators[1], signing_opts) + " 0:0:foo.txt\n" +
      ". " + Blob.sign_locator(locators[2], signing_opts) + " 0:0:foo.txt\n"

    post :create, {
      collection: {
        manifest_text: signed_manifest,
        portable_data_hash: manifest_uuid,
      }
    }
    assert_response :success
    assert_not_nil assigns(:object)
    resp = JSON.parse(@response.body)
    assert_equal manifest_uuid, resp['portable_data_hash']
    # All of the locators in the output must be signed.
    resp['manifest_text'].lines.each do |entry|
      m = /([[:xdigit:]]{32}\+\S+)/.match(entry)
      if m
        assert Blob.verify_signature m[0], signing_opts
      end
    end
  end

  test "create fails with invalid signature" do
    authorize_with :active
    signing_opts = {
      key: Rails.configuration.blob_signing_key,
      api_token: api_token(:active),
    }

    # Generate a locator with a bad signature.
    unsigned_locator = "d41d8cd98f00b204e9800998ecf8427e+0"
    bad_locator = unsigned_locator + "+Affffffff@ffffffff"
    assert !Blob.verify_signature(bad_locator, signing_opts)

    # Creating a collection with this locator should
    # produce 403 Permission denied.
    unsigned_manifest = ". #{unsigned_locator} 0:0:foo.txt\n"
    manifest_uuid = Digest::MD5.hexdigest(unsigned_manifest) +
      '+' +
      unsigned_manifest.length.to_s

    bad_manifest = ". #{bad_locator} 0:0:foo.txt\n"
    post :create, {
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
      key: Rails.configuration.blob_signing_key,
      api_token: api_token(:active),
    }

    unsigned_locator = "d41d8cd98f00b204e9800998ecf8427e+0"
    signed_locator = Blob.sign_locator(unsigned_locator, signing_opts)
    signed_manifest = ". #{signed_locator} 0:0:foo.txt\n"
    manifest_uuid = Digest::MD5.hexdigest(signed_manifest) +
      '+' +
      signed_manifest.length.to_s

    post :create, {
      collection: {
        manifest_text: signed_manifest,
        portable_data_hash: manifest_uuid
      }
    }

    assert_response 422
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
    post :create, {
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

  test "multiple signed locators per line" do
    permit_unsigned_manifests
    authorize_with :active
    locators = %w(
      d41d8cd98f00b204e9800998ecf8427e+0
      acbd18db4cc2f85cedef654fccc4a4d8+3
      ea10d51bcf88862dbcc36eb292017dfd+45)

    signing_opts = {
      key: Rails.configuration.blob_signing_key,
      api_token: api_token(:active),
    }

    unsigned_manifest = [".", *locators, "0:0:foo.txt\n"].join(" ")
    manifest_uuid = Digest::MD5.hexdigest(unsigned_manifest) +
      '+' +
      unsigned_manifest.length.to_s

    signed_locators = locators.map { |loc| Blob.sign_locator loc, signing_opts }
    signed_manifest = [".", *signed_locators, "0:0:foo.txt\n"].join(" ")

    post :create, {
      collection: {
        manifest_text: signed_manifest,
        portable_data_hash: manifest_uuid,
      }
    }
    assert_response :success
    assert_not_nil assigns(:object)
    resp = JSON.parse(@response.body)
    assert_equal manifest_uuid, resp['portable_data_hash']
    # All of the locators in the output must be signed.
    # Each line is of the form "path locator locator ... 0:0:file.txt"
    # entry.split[1..-2] will yield just the tokens in the middle of the line
    returned_locator_count = 0
    resp['manifest_text'].lines.each do |entry|
      entry.split[1..-2].each do |tok|
        returned_locator_count += 1
        assert Blob.verify_signature tok, signing_opts
      end
    end
    assert_equal locators.count, returned_locator_count
  end

  test 'Reject manifest with unsigned blob' do
    permit_unsigned_manifests false
    authorize_with :active
    unsigned_manifest = ". 0cc175b9c0f1b6a831c399e269772661+1 0:1:a.txt\n"
    manifest_uuid = Digest::MD5.hexdigest(unsigned_manifest)
    post :create, {
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
    get :index, {
      where: {name: 'expired_collection'},
    }
    assert_response :success
    found = assigns(:objects)
    assert_equal 0, found.count
  end

  test 'Show expired collection returns 404' do
    authorize_with :active
    get :show, {
      id: 'zzzzz-4zz18-mto52zx1s7sn3ih',
    }
    assert_response 404
  end

  test 'Update expired collection returns 404' do
    authorize_with :active
    post :update, {
      id: 'zzzzz-4zz18-mto52zx1s7sn3ih',
      collection: {
        name: "still expired"
      }
    }
    assert_response 404
  end

  test 'List collection with future expiration time succeeds' do
    authorize_with :active
    get :index, {
      where: {name: 'collection_expires_in_future'},
    }
    found = assigns(:objects)
    assert_equal 1, found.count
  end


  test 'Show collection with future expiration time succeeds' do
    authorize_with :active
    get :show, {
      id: 'zzzzz-4zz18-padkqo7yb8d9i3j',
    }
    assert_response :success
  end

  test 'Update collection with future expiration time succeeds' do
    authorize_with :active
    post :update, {
      id: 'zzzzz-4zz18-padkqo7yb8d9i3j',
      collection: {
        name: "still not expired"
      }
    }
    assert_response :success
  end

  test "get collection and verify that file_names is not included" do
    authorize_with :active
    get :show, {id: collections(:foo_file).uuid}
    assert_response :success
    assert_equal collections(:foo_file).uuid, json_response['uuid']
    assert_nil json_response['file_names']
    assert json_response['manifest_text']
  end

  [
    [2**8, :success],
    [2**18, 422],
  ].each do |description_size, expected_response|
    test "create collection with description size #{description_size}
          and expect response #{expected_response}" do
      authorize_with :active

      description = 'here is a collection with a very large description'
      while description.length < description_size
        description = description + description
      end

      post :create, collection: {
        manifest_text: ". d41d8cd98f00b204e9800998ecf8427e+0 0:0:foo.txt\n",
        description: description,
      }

      assert_response expected_response
    end
  end
end
