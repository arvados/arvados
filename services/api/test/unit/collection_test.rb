require 'test_helper'

class CollectionTest < ActiveSupport::TestCase
  def create_collection name, enc=nil
    txt = ". d41d8cd98f00b204e9800998ecf8427e+0 0:0:#{name}.txt\n"
    txt.force_encoding(enc) if enc
    return Collection.create(manifest_text: txt)
  end

  test 'accept ASCII manifest_text' do
    act_as_system_user do
      c = create_collection 'foo', Encoding::US_ASCII
      assert c.valid?
    end
  end

  test 'accept UTF-8 manifest_text' do
    act_as_system_user do
      c = create_collection "f\xc3\x98\xc3\x98", Encoding::UTF_8
      assert c.valid?
    end
  end

  test 'refuse manifest_text with invalid UTF-8 byte sequence' do
    act_as_system_user do
      c = create_collection "f\xc8o", Encoding::UTF_8
      assert !c.valid?
      assert_equal [:manifest_text], c.errors.messages.keys
      assert_match /UTF-8/, c.errors.messages[:manifest_text].first
    end
  end

  test 'refuse manifest_text with non-UTF-8 encoding' do
    act_as_system_user do
      c = create_collection "f\xc8o", Encoding::ASCII_8BIT
      assert !c.valid?
      assert_equal [:manifest_text], c.errors.messages.keys
      assert_match /UTF-8/, c.errors.messages[:manifest_text].first
    end
  end

  test 'create and update collection and verify file_names' do
    act_as_system_user do
      c = create_collection 'foo', Encoding::US_ASCII
      assert c.valid?
      created_file_names = c.file_names
      assert created_file_names
      assert_match /foo.txt/, c.file_names

      c.update_attribute 'manifest_text', ". d41d8cd98f00b204e9800998ecf8427e+0 0:0:foo2.txt\n"
      assert_not_equal created_file_names, c.file_names
      assert_match /foo2.txt/, c.file_names
    end
  end

  [
    [2**8, false],
    [2**18, true],
  ].each do |manifest_size, allow_truncate|
    test "create collection with manifest size #{manifest_size} with allow_truncate=#{allow_truncate},
          and not expect exceptions even on very large manifest texts" do
      # file_names has a max size, hence there will be no errors even on large manifests
      act_as_system_user do
        manifest_text = ''
        index = 0
        while manifest_text.length < manifest_size
          manifest_text += "./blurfl d41d8cd98f00b204e9800998ecf8427e+0 0:0:veryverylongfilename000000000000#{index}.txt\n"
          index += 1
        end
        manifest_text += "./laststreamname d41d8cd98f00b204e9800998ecf8427e+0 0:0:veryverylastfilename.txt\n"
        c = Collection.create(manifest_text: manifest_text)

        assert c.valid?
        assert c.file_names
        assert_match /veryverylongfilename0000000000001.txt/, c.file_names
        assert_match /veryverylongfilename0000000000002.txt/, c.file_names
        if not allow_truncate
          assert_match /veryverylastfilename/, c.file_names
          assert_match /laststreamname/, c.file_names
        end
      end
    end
  end

  test "full text search for collections" do
    # file_names column does not get populated when fixtures are loaded, hence setup test data
    act_as_system_user do
      Collection.create(manifest_text: ". acbd18db4cc2f85cedef654fccc4a4d8+3 0:3:foo\n")
      Collection.create(manifest_text: ". 37b51d194a7513e45b56f6524f2d51f2+3 0:3:bar\n")
      Collection.create(manifest_text: ". 85877ca2d7e05498dd3d109baf2df106+95+A3a4e26a366ee7e4ed3e476ccf05354761be2e4ae@545a9920 0:95:file_in_subdir1\n./subdir2/subdir3 2bbc341c702df4d8f42ec31f16c10120+64+A315d7e7bad2ce937e711fc454fae2d1194d14d64@545a9920 0:32:file1.txt 32:32:file2.txt\n./subdir2/subdir3/subdir4 2bbc341c702df4d8f42ec31f16c10120+64+A315d7e7bad2ce937e711fc454fae2d1194d14d64@545a9920 0:32:file3.txt 32:32:file4.txt\n")
    end

    [
      ['foo', true],
      ['foo bar', false],                     # no collection matching both
      ['foo&bar', false],                     # no collection matching both
      ['foo|bar', true],                      # works only no spaces between the words
      ['Gnu public', true],                   # both prefixes found, though not consecutively
      ['Gnu&public', true],                   # both prefixes found, though not consecutively
      ['file4', true],                        # prefix match
      ['file4.txt', true],                    # whole string match
      ['filex', false],                       # no such prefix
      ['subdir', true],                       # prefix matches
      ['subdir2', true],
      ['subdir2/', true],
      ['subdir2/subdir3', true],
      ['subdir2/subdir3/subdir4', true],
      ['subdir2 file4', true],                # look for both prefixes
      ['subdir4', false],                     # not a prefix match
    ].each do |search_filter, expect_results|
      search_filters = search_filter.split.each {|s| s.concat(':*')}.join('&')
      results = Collection.where("#{Collection.full_text_tsvector} @@ to_tsquery(?)",
                                 "#{search_filters}")
      if expect_results
        refute_empty results
      else
        assert_empty results
      end
    end
  end

  test 'portable data hash with missing size hints' do
    [[". d41d8cd98f00b204e9800998ecf8427e+0+Bar 0:0:x\n",
      ". d41d8cd98f00b204e9800998ecf8427e+0 0:0:x\n"],
     [". d41d8cd98f00b204e9800998ecf8427e+Foo 0:0:x\n",
      ". d41d8cd98f00b204e9800998ecf8427e 0:0:x\n"],
     [". d41d8cd98f00b204e9800998ecf8427e 0:0:x\n",
      ". d41d8cd98f00b204e9800998ecf8427e 0:0:x\n"],
    ].each do |unportable, portable|
      c = Collection.new(manifest_text: unportable)
      assert c.valid?
      assert_equal(Digest::MD5.hexdigest(portable)+"+#{portable.length}",
                   c.portable_data_hash)
    end
  end

  pdhmanifest = ". d41d8cd98f00b204e9800998ecf8427e+0 0:0:x\n"
  pdhmd5 = Digest::MD5.hexdigest pdhmanifest
  [[true, nil],
   [true, pdhmd5],
   [true, pdhmd5+'+12345'],
   [true, pdhmd5+'+'+pdhmanifest.length.to_s],
   [true, pdhmd5+'+12345+Foo'],
   [true, pdhmd5+'+Foo'],
   [false, Digest::MD5.hexdigest(pdhmanifest.strip)],
   [false, Digest::MD5.hexdigest(pdhmanifest.strip)+'+'+pdhmanifest.length.to_s],
   [false, pdhmd5[0..30]],
   [false, pdhmd5[0..30]+'z'],
   [false, pdhmd5[0..24]+'000000000'],
   [false, pdhmd5[0..24]+'000000000+0']].each do |isvalid, pdh|
    test "portable_data_hash #{pdh.inspect} valid? == #{isvalid}" do
      c = Collection.new manifest_text: pdhmanifest, portable_data_hash: pdh
      assert_equal isvalid, c.valid?, c.errors.full_messages.to_s
    end
  end

  [0, 2, 4, nil].each do |ask|
    test "set replication_desired to #{ask.inspect}" do
      Rails.configuration.default_collection_replication = 2
      act_as_user users(:active) do
        c = collections(:replication_undesired_unconfirmed)
        c.update_attributes replication_desired: ask
        assert_equal ask, c.replication_desired
      end
    end
  end

  test "replication_confirmed* can be set by admin user" do
    c = collections(:replication_desired_2_unconfirmed)
    act_as_user users(:admin) do
      assert c.update_attributes(replication_confirmed: 2,
                                 replication_confirmed_at: Time.now)
    end
  end

  test "replication_confirmed* cannot be set by non-admin user" do
    act_as_user users(:active) do
      c = collections(:replication_desired_2_unconfirmed)
      # Cannot set just one at a time.
      assert_raise ArvadosModel::PermissionDeniedError do
        c.update_attributes replication_confirmed: 1
      end
      assert_raise ArvadosModel::PermissionDeniedError do
        c.update_attributes replication_confirmed_at: Time.now
      end
      # Cannot set both at once, either.
      assert_raise ArvadosModel::PermissionDeniedError do
        c.update_attributes(replication_confirmed: 1,
                            replication_confirmed_at: Time.now)
      end
    end
  end

  test "replication_confirmed* can be cleared (but only together) by non-admin user" do
    act_as_user users(:active) do
      c = collections(:replication_desired_2_confirmed_2)
      # Cannot clear just one at a time.
      assert_raise ArvadosModel::PermissionDeniedError do
        c.update_attributes replication_confirmed: nil
      end
      c.reload
      assert_raise ArvadosModel::PermissionDeniedError do
        c.update_attributes replication_confirmed_at: nil
      end
      # Can clear both at once.
      c.reload
      assert c.update_attributes(replication_confirmed: nil,
                                 replication_confirmed_at: nil)
    end
  end

  test "clear replication_confirmed* when introducing a new block in manifest" do
    c = collections(:replication_desired_2_confirmed_2)
    act_as_user users(:active) do
      assert c.update_attributes(manifest_text: collections(:user_agreement).signed_manifest_text)
      assert_nil c.replication_confirmed
      assert_nil c.replication_confirmed_at
    end
  end

  test "don't clear replication_confirmed* when just renaming a file" do
    c = collections(:replication_desired_2_confirmed_2)
    act_as_user users(:active) do
      new_manifest = c.signed_manifest_text.sub(':bar', ':foo')
      assert c.update_attributes(manifest_text: new_manifest)
      assert_equal 2, c.replication_confirmed
      assert_not_nil c.replication_confirmed_at
    end
  end

  test "don't clear replication_confirmed* when just deleting a data block" do
    c = collections(:replication_desired_2_confirmed_2)
    act_as_user users(:active) do
      new_manifest = c.signed_manifest_text
      new_manifest.sub!(/ \S+:bar/, '')
      new_manifest.sub!(/ acbd\S+/, '')

      # Confirm that we did just remove a block from the manifest (if
      # not, this test would pass without testing the relevant case):
      assert_operator new_manifest.length+40, :<, c.signed_manifest_text.length

      assert c.update_attributes(manifest_text: new_manifest)
      assert_equal 2, c.replication_confirmed
      assert_not_nil c.replication_confirmed_at
    end
  end

  test "create collection with properties" do
    act_as_system_user do
      c = Collection.create(manifest_text: ". acbd18db4cc2f85cedef654fccc4a4d8+3 0:3:foo\n",
                            properties: {'property_1' => 'value_1'})
      assert c.valid?
      assert_equal 'value_1', c.properties['property_1']
    end
  end

  test 'create, delete, recreate collection with same name and owner' do
    act_as_user users(:active) do
      # create collection with name
      c = Collection.create(manifest_text: '',
                            name: "test collection name")
      assert c.valid?
      uuid = c.uuid

      # mark collection as expired
      c.update_attribute 'expires_at', Time.new.strftime("%Y-%m-%d")
      c = Collection.where(uuid: uuid)
      assert_empty c, 'Should not be able to find expired collection'

      # recreate collection with the same name
      c = Collection.create(manifest_text: '',
                            name: "test collection name")
      assert c.valid?
    end
  end

  test "find_all_for_docker_image resolves names that look like hashes" do
    coll_list = Collection.
      find_all_for_docker_image('a' * 64, nil, [users(:active)])
    coll_uuids = coll_list.map(&:uuid)
    assert_includes(coll_uuids, collections(:docker_image).uuid)
  end
end
