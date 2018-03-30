# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

require 'test_helper'
require 'sweep_trashed_objects'

class CollectionTest < ActiveSupport::TestCase
  include DbCurrentTime

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
      assert_match(/UTF-8/, c.errors.messages[:manifest_text].first)
    end
  end

  test 'refuse manifest_text with non-UTF-8 encoding' do
    act_as_system_user do
      c = create_collection "f\xc8o", Encoding::ASCII_8BIT
      assert !c.valid?
      assert_equal [:manifest_text], c.errors.messages.keys
      assert_match(/UTF-8/, c.errors.messages[:manifest_text].first)
    end
  end

  [
    ". 0:0:foo.txt",
    ". d41d8cd98f00b204e9800998ecf8427e foo.txt",
    "d41d8cd98f00b204e9800998ecf8427e 0:0:foo.txt",
    ". d41d8cd98f00b204e9800998ecf8427e 0:0:foo.txt",
  ].each do |manifest_text|
    test "create collection with invalid manifest text #{manifest_text} and expect error" do
      act_as_system_user do
        c = Collection.create(manifest_text: manifest_text)
        assert !c.valid?
      end
    end
  end

  [
    nil,
    "",
    ". d41d8cd98f00b204e9800998ecf8427e 0:0:foo.txt\n",
  ].each do |manifest_text|
    test "create collection with valid manifest text #{manifest_text.inspect} and expect success" do
      act_as_system_user do
        c = Collection.create(manifest_text: manifest_text)
        assert c.valid?
      end
    end
  end

  [
    ". 0:0:foo.txt",
    ". d41d8cd98f00b204e9800998ecf8427e foo.txt",
    "d41d8cd98f00b204e9800998ecf8427e 0:0:foo.txt",
    ". d41d8cd98f00b204e9800998ecf8427e 0:0:foo.txt",
  ].each do |manifest_text|
    test "update collection with invalid manifest text #{manifest_text} and expect error" do
      act_as_system_user do
        c = create_collection 'foo', Encoding::US_ASCII
        assert c.valid?

        c.update_attribute 'manifest_text', manifest_text
        assert !c.valid?
      end
    end
  end

  [
    nil,
    "",
    ". d41d8cd98f00b204e9800998ecf8427e 0:0:foo.txt\n",
  ].each do |manifest_text|
    test "update collection with valid manifest text #{manifest_text.inspect} and expect success" do
      act_as_system_user do
        c = create_collection 'foo', Encoding::US_ASCII
        assert c.valid?

        c.update_attribute 'manifest_text', manifest_text
        assert c.valid?
      end
    end
  end

  test 'create and update collection and verify file_names' do
    act_as_system_user do
      c = create_collection 'foo', Encoding::US_ASCII
      assert c.valid?
      created_file_names = c.file_names
      assert created_file_names
      assert_match(/foo.txt/, c.file_names)

      c.update_attribute 'manifest_text', ". d41d8cd98f00b204e9800998ecf8427e+0 0:0:foo2.txt\n"
      assert_not_equal created_file_names, c.file_names
      assert_match(/foo2.txt/, c.file_names)
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
        assert_match(/veryverylongfilename0000000000001.txt/, c.file_names)
        assert_match(/veryverylongfilename0000000000002.txt/, c.file_names)
        if not allow_truncate
          assert_match(/veryverylastfilename/, c.file_names)
          assert_match(/laststreamname/, c.file_names)
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

  test "storage_classes_desired cannot be empty" do
    act_as_user users(:active) do
      c = collections(:collection_owned_by_active)
      c.update_attributes storage_classes_desired: ["hot"]
      assert_equal ["hot"], c.storage_classes_desired
      assert_raise ArvadosModel::InvalidStateTransitionError do
        c.update_attributes storage_classes_desired: []
      end
    end
  end

  test "storage classes lists should only contain non-empty strings" do
    c = collections(:storage_classes_desired_default_unconfirmed)
    act_as_user users(:admin) do
      assert c.update_attributes(storage_classes_desired: ["default", "a_string"],
                                 storage_classes_confirmed: ["another_string"])
      [
        ["storage_classes_desired", ["default", 42]],
        ["storage_classes_confirmed", [{the_answer: 42}]],
        ["storage_classes_desired", ["default", ""]],
        ["storage_classes_confirmed", [""]],
      ].each do |attr, val|
        assert_raise ArvadosModel::InvalidStateTransitionError do
          assert c.update_attributes({attr => val})
        end
      end
    end
  end

  test "storage_classes_confirmed* can be set by admin user" do
    c = collections(:storage_classes_desired_default_unconfirmed)
    act_as_user users(:admin) do
      assert c.update_attributes(storage_classes_confirmed: ["default"],
                                 storage_classes_confirmed_at: Time.now)
    end
  end

  test "storage_classes_confirmed* cannot be set by non-admin user" do
    act_as_user users(:active) do
      c = collections(:storage_classes_desired_default_unconfirmed)
      # Cannot set just one at a time.
      assert_raise ArvadosModel::PermissionDeniedError do
        c.update_attributes storage_classes_confirmed: ["default"]
      end
      c.reload
      assert_raise ArvadosModel::PermissionDeniedError do
        c.update_attributes storage_classes_confirmed_at: Time.now
      end
      # Cannot set bot at once, either.
      c.reload
      assert_raise ArvadosModel::PermissionDeniedError do
        assert c.update_attributes(storage_classes_confirmed: ["default"],
                                   storage_classes_confirmed_at: Time.now)
      end
    end
  end

  test "storage_classes_confirmed* can be cleared (but only together) by non-admin user" do
    act_as_user users(:active) do
      c = collections(:storage_classes_desired_default_confirmed_default)
      # Cannot clear just one at a time.
      assert_raise ArvadosModel::PermissionDeniedError do
        c.update_attributes storage_classes_confirmed: []
      end
      c.reload
      assert_raise ArvadosModel::PermissionDeniedError do
        c.update_attributes storage_classes_confirmed_at: nil
      end
      # Can clear both at once.
      c.reload
      assert c.update_attributes(storage_classes_confirmed: [],
                                 storage_classes_confirmed_at: nil)
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

  test 'signature expiry does not exceed trash_at' do
    act_as_user users(:active) do
      t0 = db_current_time
      c = Collection.create!(manifest_text: ". d41d8cd98f00b204e9800998ecf8427e+0 0:0:x\n", name: 'foo')
      c.update_attributes! trash_at: (t0 + 1.hours)
      c.reload
      sig_exp = /\+A[0-9a-f]{40}\@([0-9]+)/.match(c.signed_manifest_text)[1].to_i
      assert_operator sig_exp.to_i, :<=, (t0 + 1.hours).to_i
    end
  end

  test 'far-future expiry date cannot be used to circumvent configured permission ttl' do
    act_as_user users(:active) do
      c = Collection.create!(manifest_text: ". d41d8cd98f00b204e9800998ecf8427e+0 0:0:x\n",
                             name: 'foo',
                             trash_at: db_current_time + 1.years)
      sig_exp = /\+A[0-9a-f]{40}\@([0-9]+)/.match(c.signed_manifest_text)[1].to_i
      expect_max_sig_exp = db_current_time.to_i + Rails.configuration.blob_signature_ttl
      assert_operator c.trash_at.to_i, :>, expect_max_sig_exp
      assert_operator sig_exp.to_i, :<=, expect_max_sig_exp
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

      c = Collection.readable_by(current_user).where(uuid: uuid)
      assert_not_empty c, 'Should be able to find live collection'

      # mark collection as expired
      c.first.update_attributes!(trash_at: Time.new.strftime("%Y-%m-%d"))
      c = Collection.readable_by(current_user).where(uuid: uuid)
      assert_empty c, 'Should not be able to find expired collection'

      # recreate collection with the same name
      c = Collection.create(manifest_text: '',
                            name: "test collection name")
      assert c.valid?
    end
  end

  test 'trash_at cannot be set too far in the past' do
    act_as_user users(:active) do
      t0 = db_current_time
      c = Collection.create!(manifest_text: '', name: 'foo')
      c.update_attributes! trash_at: (t0 - 2.weeks)
      c.reload
      assert_operator c.trash_at, :>, t0
    end
  end

  now = Time.now
  [['trash-to-delete interval negative',
    :collection_owned_by_active,
    {trash_at: now+2.weeks, delete_at: now},
    {state: :invalid}],
   ['now-to-delete interval short',
    :collection_owned_by_active,
    {trash_at: now+3.days, delete_at: now+7.days},
    {state: :trash_future}],
   ['now-to-delete interval short, trash=delete',
    :collection_owned_by_active,
    {trash_at: now+3.days, delete_at: now+3.days},
    {state: :trash_future}],
   ['trash-to-delete interval ok',
    :collection_owned_by_active,
    {trash_at: now, delete_at: now+15.days},
    {state: :trash_now}],
   ['trash-to-delete interval short, but far enough in future',
    :collection_owned_by_active,
    {trash_at: now+13.days, delete_at: now+15.days},
    {state: :trash_future}],
   ['trash by setting is_trashed bool',
    :collection_owned_by_active,
    {is_trashed: true},
    {state: :trash_now}],
   ['trash in future by setting just trash_at',
    :collection_owned_by_active,
    {trash_at: now+1.week},
    {state: :trash_future}],
   ['trash in future by setting trash_at and delete_at',
    :collection_owned_by_active,
    {trash_at: now+1.week, delete_at: now+4.weeks},
    {state: :trash_future}],
   ['untrash by clearing is_trashed bool',
    :expired_collection,
    {is_trashed: false},
    {state: :not_trash}],
  ].each do |test_name, fixture_name, updates, expect|
    test test_name do
      act_as_user users(:active) do
        min_exp = (db_current_time +
                   Rails.configuration.blob_signature_ttl.seconds)
        if fixture_name == :expired_collection
          # Fixture-finder shorthand doesn't find trashed collections
          # because they're not in the default scope.
          c = Collection.find_by_uuid('zzzzz-4zz18-mto52zx1s7sn3ih')
        else
          c = collections(fixture_name)
        end
        updates_ok = c.update_attributes(updates)
        expect_valid = expect[:state] != :invalid
        assert_equal expect_valid, updates_ok, c.errors.full_messages.to_s
        case expect[:state]
        when :invalid
          refute c.valid?
        when :trash_now
          assert c.is_trashed
          assert_not_nil c.trash_at
          assert_operator c.trash_at, :<=, db_current_time
          assert_not_nil c.delete_at
          assert_operator c.delete_at, :>=, min_exp
        when :trash_future
          refute c.is_trashed
          assert_not_nil c.trash_at
          assert_operator c.trash_at, :>, db_current_time
          assert_not_nil c.delete_at
          assert_operator c.delete_at, :>=, c.trash_at
          # Currently this minimum interval is needed to prevent early
          # garbage collection:
          assert_operator c.delete_at, :>=, min_exp
        when :not_trash
          refute c.is_trashed
          assert_nil c.trash_at
          assert_nil c.delete_at
        else
          raise "bad expect[:state]==#{expect[:state].inspect} in test case"
        end
      end
    end
  end

  test 'default trash interval > blob signature ttl' do
    Rails.configuration.default_trash_lifetime = 86400 * 21 # 3 weeks
    start = db_current_time
    act_as_user users(:active) do
      c = Collection.create!(manifest_text: '', name: 'foo')
      c.update_attributes!(trash_at: start + 86400.seconds)
      assert_operator c.delete_at, :>=, start + (86400*22).seconds
      assert_operator c.delete_at, :<, start + (86400*22 + 30).seconds
      c.destroy

      c = Collection.create!(manifest_text: '', name: 'foo')
      c.update_attributes!(is_trashed: true)
      assert_operator c.delete_at, :>=, start + (86400*21).seconds
    end
  end

  test "find_all_for_docker_image resolves names that look like hashes" do
    coll_list = Collection.
      find_all_for_docker_image('a' * 64, nil, [users(:active)])
    coll_uuids = coll_list.map(&:uuid)
    assert_includes(coll_uuids, collections(:docker_image).uuid)
  end

  test "move collections to trash in SweepTrashedObjects" do
    c = collections(:trashed_on_next_sweep)
    refute_empty Collection.where('uuid=? and is_trashed=false', c.uuid)
    assert_raises(ActiveRecord::RecordNotUnique) do
      act_as_user users(:active) do
        Collection.create!(owner_uuid: c.owner_uuid,
                           name: c.name)
      end
    end
    SweepTrashedObjects.sweep_now
    c = Collection.where('uuid=? and is_trashed=true', c.uuid).first
    assert c
    act_as_user users(:active) do
      assert Collection.create!(owner_uuid: c.owner_uuid,
                                name: c.name)
    end
  end

  test "delete collections in SweepTrashedObjects" do
    uuid = 'zzzzz-4zz18-3u1p5umicfpqszp' # deleted_on_next_sweep
    assert_not_empty Collection.where(uuid: uuid)
    SweepTrashedObjects.sweep_now
    assert_empty Collection.where(uuid: uuid)
  end

  test "delete referring links in SweepTrashedObjects" do
    uuid = collections(:trashed_on_next_sweep).uuid
    act_as_system_user do
      Link.create!(head_uuid: uuid,
                   tail_uuid: system_user_uuid,
                   link_class: 'whatever',
                   name: 'something')
    end
    past = db_current_time
    Collection.where(uuid: uuid).
      update_all(is_trashed: true, trash_at: past, delete_at: past)
    assert_not_empty Collection.where(uuid: uuid)
    SweepTrashedObjects.sweep_now
    assert_empty Collection.where(uuid: uuid)
  end
end
