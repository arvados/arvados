# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

require 'test_helper'
require 'sweep_trashed_objects'
require 'fix_collection_versions_timestamps'

class CollectionTest < ActiveSupport::TestCase
  include DbCurrentTime

  def create_collection name, enc=nil
    txt = ". d41d8cd98f00b204e9800998ecf8427e+0 0:0:#{name}.txt\n"
    txt.force_encoding(enc) if enc
    return Collection.create(manifest_text: txt, name: name)
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
    [". d41d8cd98f00b204e9800998ecf8427e 0:34:foo.txt\n", 1, 34],
    [". d41d8cd98f00b204e9800998ecf8427e 0:34:foo.txt 0:30:foo.txt 0:30:foo1.txt 0:30:foo2.txt 0:30:foo3.txt 0:30:foo4.txt\n", 5, 184],
    [". d41d8cd98f00b204e9800998ecf8427e 0:0:.\n", 0, 0]
  ].each do |manifest, count, size|
    test "file stats on create collection with #{manifest}" do
      act_as_system_user do
        c = Collection.create(manifest_text: manifest)
        assert_equal count, c.file_count
        assert_equal size, c.file_size_total
      end
    end
  end

  test "file stats cannot be changed unless through manifest change" do
    act_as_system_user do
      # Direct changes to file stats should be ignored
      c = Collection.create(manifest_text: ". d41d8cd98f00b204e9800998ecf8427e 0:34:foo.txt\n")
      c.file_count = 6
      c.file_size_total = 30
      assert c.valid?
      assert_equal 1, c.file_count
      assert_equal 34, c.file_size_total

      # File stats specified on create should be ignored and overwritten
      c = Collection.create(manifest_text: ". d41d8cd98f00b204e9800998ecf8427e 0:34:foo.txt\n", file_count: 10, file_size_total: 10)
      assert c.valid?
      assert_equal 1, c.file_count
      assert_equal 34, c.file_size_total

      # Updating the manifest should change file stats
      c.update_attributes(manifest_text: ". d41d8cd98f00b204e9800998ecf8427e 0:34:foo.txt 0:34:foo2.txt\n")
      assert c.valid?
      assert_equal 2, c.file_count
      assert_equal 68, c.file_size_total

      # Updating file stats and the manifest should use manifest values
      c.update_attributes(manifest_text: ". d41d8cd98f00b204e9800998ecf8427e 0:34:foo.txt\n", file_count:10, file_size_total: 10)
      assert c.valid?
      assert_equal 1, c.file_count
      assert_equal 34, c.file_size_total

      # Updating just the file stats should be ignored
      c.update_attributes(file_count: 10, file_size_total: 10)
      assert c.valid?
      assert_equal 1, c.file_count
      assert_equal 34, c.file_size_total
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

  test "auto-create version after idle setting" do
    Rails.configuration.Collections.CollectionVersioning = true
    Rails.configuration.Collections.PreserveVersionIfIdle = 600 # 10 minutes
    act_as_user users(:active) do
      # Set up initial collection
      c = create_collection 'foo', Encoding::US_ASCII
      assert c.valid?
      assert_equal 1, c.version
      assert_equal false, c.preserve_version
      # Make a versionable update, it shouldn't create a new version yet
      c.update_attributes!({'name' => 'bar'})
      c.reload
      assert_equal 'bar', c.name
      assert_equal 1, c.version
      # Update modified_at to trigger a version auto-creation
      fifteen_min_ago = Time.now - 15.minutes
      c.update_column('modified_at', fifteen_min_ago) # Update without validations/callbacks
      c.reload
      assert_equal fifteen_min_ago.to_i, c.modified_at.to_i
      c.update_attributes!({'name' => 'baz'})
      c.reload
      assert_equal 'baz', c.name
      assert_equal 2, c.version
      # Make another update, no new version should be created
      c.update_attributes!({'name' => 'foobar'})
      c.reload
      assert_equal 'foobar', c.name
      assert_equal 2, c.version
    end
  end

  test "preserve_version updates" do
    Rails.configuration.Collections.CollectionVersioning = true
    Rails.configuration.Collections.PreserveVersionIfIdle = -1 # disabled
    act_as_user users(:active) do
      # Set up initial collection
      c = create_collection 'foo', Encoding::US_ASCII
      assert c.valid?
      assert_equal 1, c.version
      assert_equal false, c.preserve_version
      # This update shouldn't produce a new version, as the idle time is not up
      c.update_attributes!({
        'name' => 'bar'
      })
      c.reload
      assert_equal 1, c.version
      assert_equal 'bar', c.name
      assert_equal false, c.preserve_version
      # This update should produce a new version, even if the idle time is not up
      # and also keep the preserve_version=true flag to persist it.
      c.update_attributes!({
        'name' => 'baz',
        'preserve_version' => true
      })
      c.reload
      assert_equal 2, c.version
      assert_equal 'baz', c.name
      assert_equal true, c.preserve_version
      # Make sure preserve_version is not disabled after being enabled, unless
      # a new version is created.
      # This is a non-versionable update
      c.update_attributes!({
        'preserve_version' => false,
        'replication_desired' => 2
      })
      c.reload
      assert_equal 2, c.version
      assert_equal 2, c.replication_desired
      assert_equal true, c.preserve_version
      # This is a versionable update
      c.update_attributes!({
        'preserve_version' => false,
        'name' => 'foobar'
      })
      c.reload
      assert_equal 3, c.version
      assert_equal false, c.preserve_version
      assert_equal 'foobar', c.name
      # Flipping only 'preserve_version' to true doesn't create a new version
      c.update_attributes!({'preserve_version' => true})
      c.reload
      assert_equal 3, c.version
      assert_equal true, c.preserve_version
    end
  end

  test "preserve_version updates don't change modified_at timestamp" do
    act_as_user users(:active) do
      c = create_collection 'foo', Encoding::US_ASCII
      assert c.valid?
      assert_equal false, c.preserve_version
      modified_at = c.modified_at.to_f
      c.update_attributes!({'preserve_version' => true})
      c.reload
      assert_equal true, c.preserve_version
      assert_equal modified_at, c.modified_at.to_f,
        'preserve_version updates should not trigger modified_at changes'
    end
  end

  [
    ['version', 10],
    ['current_version_uuid', 'zzzzz-4zz18-bv31uwvy3neko21'],
  ].each do |name, new_value|
    test "'#{name}' updates on current version collections are not allowed" do
      act_as_user users(:active) do
        # Set up initial collection
        c = create_collection 'foo', Encoding::US_ASCII
        assert c.valid?
        assert_equal 1, c.version

        assert_raises(ActiveRecord::RecordInvalid) do
          c.update_attributes!({
            name => new_value
          })
        end
      end
    end
  end

  test "uuid updates on current version make older versions update their pointers" do
    Rails.configuration.Collections.CollectionVersioning = true
    Rails.configuration.Collections.PreserveVersionIfIdle = 0
    act_as_system_user do
      # Set up initial collection
      c = create_collection 'foo', Encoding::US_ASCII
      assert c.valid?
      assert_equal 1, c.version
      # Make changes so that a new version is created
      c.update_attributes!({'name' => 'bar'})
      c.reload
      assert_equal 2, c.version
      assert_equal 2, Collection.where(current_version_uuid: c.uuid).count
      new_uuid = 'zzzzz-4zz18-somefakeuuidnow'
      assert_empty Collection.where(uuid: new_uuid)
      # Update UUID on current version, check that both collections point to it
      c.update_attributes!({'uuid' => new_uuid})
      c.reload
      assert_equal new_uuid, c.uuid
      assert_equal 2, Collection.where(current_version_uuid: new_uuid).count
    end
  end

  # This test exposes a bug related to JSONB attributes, see #15725.
  test "recently loaded collection shouldn't list changed attributes" do
    col = Collection.where("properties != '{}'::jsonb").limit(1).first
    refute col.properties_changed?, 'Properties field should not be seen as changed'
  end

  [
    [
      true,
      {'foo'=>'bar', 'lst'=>[1, 3, 5, 7], 'hsh'=>{'baz'=>'qux', 'foobar'=>true, 'hsh'=>{'nested'=>true}}, 'delete_at'=>nil},
      {:foo=>:bar, :lst=>[1, 3, 5, 7], :hsh=>{'baz'=>'qux', :foobar=>true, 'hsh'=>{:nested=>true}}, :delete_at=>nil},
    ],
    [
      true,
      {'foo'=>'bar', 'lst'=>[1, 3, 5, 7], 'hsh'=>{'baz'=>'qux', 'foobar'=>true, 'hsh'=>{'nested'=>true}}, 'delete_at'=>nil},
      {'delete_at'=>nil, 'foo'=>'bar', 'lst'=>[1, 3, 5, 7], 'hsh'=>{'baz'=>'qux', 'foobar'=>true, 'hsh'=>{'nested'=>true}}},
    ],
    [
      true,
      {'foo'=>'bar', 'lst'=>[1, 3, 5, 7], 'hsh'=>{'baz'=>'qux', 'foobar'=>true, 'hsh'=>{'nested'=>true}}, 'delete_at'=>nil},
      {'delete_at'=>nil, 'foo'=>'bar', 'lst'=>[1, 3, 5, 7], 'hsh'=>{'foobar'=>true, 'hsh'=>{'nested'=>true}, 'baz'=>'qux'}},
    ],
    [
      false,
      {'foo'=>'bar', 'lst'=>[1, 3, 5, 7], 'hsh'=>{'baz'=>'qux', 'foobar'=>true, 'hsh'=>{'nested'=>true}}, 'delete_at'=>nil},
      {'foo'=>'bar', 'lst'=>[1, 3, 5, 42], 'hsh'=>{'baz'=>'qux', 'foobar'=>true, 'hsh'=>{'nested'=>true}}, 'delete_at'=>nil},
    ],
    [
      false,
      {'foo'=>'bar', 'lst'=>[1, 3, 5, 7], 'hsh'=>{'baz'=>'qux', 'foobar'=>true, 'hsh'=>{'nested'=>true}}, 'delete_at'=>nil},
      {'foo'=>'bar', 'lst'=>[1, 3, 7, 5], 'hsh'=>{'baz'=>'qux', 'foobar'=>true, 'hsh'=>{'nested'=>true}}, 'delete_at'=>nil},
    ],
    [
      false,
      {'foo'=>'bar', 'lst'=>[1, 3, 5, 7], 'hsh'=>{'baz'=>'qux', 'foobar'=>true, 'hsh'=>{'nested'=>true}}, 'delete_at'=>nil},
      {'foo'=>'bar', 'lst'=>[1, 3, 5, 7], 'hsh'=>{'baz'=>'qux', 'foobar'=>true, 'hsh'=>{'nested'=>false}}, 'delete_at'=>nil},
    ],
    [
      false,
      {'foo'=>'bar', 'lst'=>[1, 3, 5, 7], 'hsh'=>{'baz'=>'qux', 'foobar'=>true, 'hsh'=>{'nested'=>true}}, 'delete_at'=>nil},
      {'foo'=>'bar', 'lst'=>[1, 3, 5, 7], 'hsh'=>{'baz'=>'qux', 'foobar'=>true, 'hsh'=>{'nested'=>true}}, 'delete_at'=>1234567890},
    ],
  ].each do |should_be_equal, value_1, value_2|
    test "JSONB properties #{value_1} is#{should_be_equal ? '' : ' not'} equal to #{value_2}" do
      act_as_user users(:active) do
        # Set up initial collection
        c = create_collection 'foo', Encoding::US_ASCII
        assert c.valid?
        c.update_attributes!({'properties' => value_1})
        c.reload
        assert c.changes.keys.empty?
        c.properties = value_2
        if should_be_equal
          assert c.changes.keys.empty?, "Properties #{value_1.inspect} should be equal to #{value_2.inspect}"
        else
          refute c.changes.keys.empty?, "Properties #{value_1.inspect} should not be equal to #{value_2.inspect}"
        end
      end
    end
  end

  test "older versions' modified_at indicate when they're created" do
    Rails.configuration.Collections.CollectionVersioning = true
    Rails.configuration.Collections.PreserveVersionIfIdle = 0
    act_as_user users(:active) do
      # Set up initial collection
      c = create_collection 'foo', Encoding::US_ASCII
      assert c.valid?
      original_version_modified_at = c.modified_at.to_f
      # Make changes so that a new version is created
      c.update_attributes!({'name' => 'bar'})
      c.reload
      assert_equal 2, c.version
      # Get the old version
      c_old = Collection.where(current_version_uuid: c.uuid, version: 1).first
      assert_not_nil c_old

      version_creation_datetime = c_old.modified_at.to_f
      assert_equal c.created_at.to_f, c_old.created_at.to_f
      assert_equal original_version_modified_at, version_creation_datetime

      # Make update on current version so old version get the attribute synced;
      # its modified_at should not change.
      new_replication = 3
      c.update_attributes!({'replication_desired' => new_replication})
      c.reload
      assert_equal new_replication, c.replication_desired
      c_old.reload
      assert_equal new_replication, c_old.replication_desired
      assert_equal version_creation_datetime, c_old.modified_at.to_f
      assert_operator c.modified_at.to_f, :>, c_old.modified_at.to_f
    end
  end

  # Bug #17152 - This test relies on fixtures simulating the problem.
  test "migration fixing collection versions' modified_at timestamps" do
    versioned_collection_fixtures = [
      collections(:w_a_z_file).uuid,
      collections(:collection_owned_by_active).uuid
    ]
    versioned_collection_fixtures.each do |uuid|
      cols = Collection.where(current_version_uuid: uuid).order(version: :desc)
      assert_equal cols.size, 2
      # cols[0] -> head version // cols[1] -> old version
      assert_operator (cols[0].modified_at.to_f - cols[1].modified_at.to_f), :==, 0
      assert cols[1].modified_at != cols[1].created_at
    end
    fix_collection_versions_timestamps
    versioned_collection_fixtures.each do |uuid|
      cols = Collection.where(current_version_uuid: uuid).order(version: :desc)
      assert_equal cols.size, 2
      # cols[0] -> head version // cols[1] -> old version
      assert_operator (cols[0].modified_at.to_f - cols[1].modified_at.to_f), :>, 1
      assert_operator cols[1].modified_at, :==, cols[1].created_at
    end
  end

  test "past versions should not be directly updatable" do
    Rails.configuration.Collections.CollectionVersioning = true
    Rails.configuration.Collections.PreserveVersionIfIdle = 0
    act_as_system_user do
      # Set up initial collection
      c = create_collection 'foo', Encoding::US_ASCII
      assert c.valid?
      # Make changes so that a new version is created
      c.update_attributes!({'name' => 'bar'})
      c.reload
      assert_equal 2, c.version
      # Get the old version
      c_old = Collection.where(current_version_uuid: c.uuid, version: 1).first
      assert_not_nil c_old
      # With collection versioning still being enabled, try to update
      c_old.name = 'this was foo'
      assert c_old.invalid?
      c_old.reload
      # Try to fool the validator attempting to make c_old to look like a
      # current version, it should also fail.
      c_old.current_version_uuid = c_old.uuid
      assert c_old.invalid?
      c_old.reload
      # Now disable collection versioning, it should behave the same way
      Rails.configuration.Collections.CollectionVersioning = false
      c_old.name = 'this was foo'
      assert c_old.invalid?
    end
  end

  [
    ['owner_uuid', 'zzzzz-tpzed-d9tiejq69daie8f', 'zzzzz-tpzed-xurymjxw79nv3jz'],
    ['replication_desired', 2, 3],
    ['storage_classes_desired', ['hot'], ['archive']],
  ].each do |attr, first_val, second_val|
    test "sync #{attr} with older versions" do
      Rails.configuration.Collections.CollectionVersioning = true
      Rails.configuration.Collections.PreserveVersionIfIdle = 0
      act_as_system_user do
        # Set up initial collection
        c = create_collection 'foo', Encoding::US_ASCII
        assert c.valid?
        assert_equal 1, c.version
        assert_not_equal first_val, c.attributes[attr]
        # Make changes so that a new version is created and a synced field is
        # updated on both
        c.update_attributes!({'name' => 'bar', attr => first_val})
        c.reload
        assert_equal 2, c.version
        assert_equal first_val, c.attributes[attr]
        assert_equal 2, Collection.where(current_version_uuid: c.uuid).count
        assert_equal first_val, Collection.where(current_version_uuid: c.uuid, version: 1).first.attributes[attr]
        # Only make an update on the same synced field & check that the previously
        # created version also gets it.
        c.update_attributes!({attr => second_val})
        c.reload
        assert_equal 2, c.version
        assert_equal second_val, c.attributes[attr]
        assert_equal 2, Collection.where(current_version_uuid: c.uuid).count
        assert_equal second_val, Collection.where(current_version_uuid: c.uuid, version: 1).first.attributes[attr]
      end
    end
  end

  [
    [false, 'name', 'bar', false],
    [false, 'description', 'The quick brown fox jumps over the lazy dog', false],
    [false, 'properties', {'new_version' => true}, false],
    [false, 'manifest_text', ". d41d8cd98f00b204e9800998ecf8427e 0:0:foo.txt\n", false],
    [true, 'name', 'bar', true],
    [true, 'description', 'The quick brown fox jumps over the lazy dog', true],
    [true, 'properties', {'new_version' => true}, true],
    [true, 'manifest_text', ". d41d8cd98f00b204e9800998ecf8427e 0:0:foo.txt\n", true],
    # Non-versionable attribute updates shouldn't create new versions
    [true, 'replication_desired', 5, false],
    [false, 'replication_desired', 5, false],
  ].each do |versioning, attr, val, new_version_expected|
    test "update #{attr} with versioning #{versioning ? '' : 'not '}enabled should #{new_version_expected ? '' : 'not '}create a new version" do
      Rails.configuration.Collections.CollectionVersioning = versioning
      Rails.configuration.Collections.PreserveVersionIfIdle = 0
      act_as_user users(:active) do
        # Create initial collection
        c = create_collection 'foo', Encoding::US_ASCII
        assert c.valid?
        assert_equal 'foo', c.name

        # Check current version attributes
        assert_equal 1, c.version
        assert_equal c.uuid, c.current_version_uuid

        # Update attribute and check if version number should be incremented
        old_value = c.attributes[attr]
        c.update_attributes!({attr => val})
        assert_equal new_version_expected, c.version == 2
        assert_equal val, c.attributes[attr]

        if versioning && new_version_expected
          # Search for the snapshot & previous value
          assert_equal 2, Collection.where(current_version_uuid: c.uuid).count
          s = Collection.where(current_version_uuid: c.uuid, version: 1).first
          assert_not_nil s
          assert_equal old_value, s.attributes[attr]
        else
          # If versioning is disabled or no versionable attribute was updated,
          # only the current version should exist
          assert_equal 1, Collection.where(current_version_uuid: c.uuid).count
          assert_equal c, Collection.where(current_version_uuid: c.uuid).first
        end
      end
    end
  end

  test 'current_version_uuid is ignored during update' do
    Rails.configuration.Collections.CollectionVersioning = true
    Rails.configuration.Collections.PreserveVersionIfIdle = 0
    act_as_user users(:active) do
      # Create 1st collection
      col1 = create_collection 'foo', Encoding::US_ASCII
      assert col1.valid?
      assert_equal 1, col1.version

      # Create 2nd collection, update it so it becomes version:2
      # (to avoid unique index violation)
      col2 = create_collection 'bar', Encoding::US_ASCII
      assert col2.valid?
      assert_equal 1, col2.version
      col2.update_attributes({name: 'baz'})
      assert_equal 2, col2.version

      # Try to make col2 a past version of col1. It shouldn't be possible
      col2.update_attributes({current_version_uuid: col1.uuid})
      assert col2.invalid?
      col2.reload
      assert_not_equal col1.uuid, col2.current_version_uuid
    end
  end

  test 'with versioning enabled, simultaneous updates increment version correctly' do
    Rails.configuration.Collections.CollectionVersioning = true
    Rails.configuration.Collections.PreserveVersionIfIdle = 0
    act_as_user users(:active) do
      # Create initial collection
      col = create_collection 'foo', Encoding::US_ASCII
      assert col.valid?
      assert_equal 1, col.version

      # Simulate simultaneous updates
      c1 = Collection.where(uuid: col.uuid).first
      assert_equal 1, c1.version
      c1.name = 'bar'
      c2 = Collection.where(uuid: col.uuid).first
      c2.description = 'foo collection'
      c1.save!
      assert_equal 1, c2.version
      # with_lock forces a reload, so this shouldn't produce an unique violation error
      c2.save!
      assert_equal 3, c2.version
      assert_equal 'foo collection', c2.description
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

  test "storage_classes_desired default respects config" do
    saved = Rails.configuration.DefaultStorageClasses
    Rails.configuration.DefaultStorageClasses = ["foo"]
    begin
      act_as_user users(:active) do
        c = Collection.create!
        assert_equal ["foo"], c.storage_classes_desired
      end
    ensure
      Rails.configuration.DefaultStorageClasses = saved
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
      Rails.configuration.Collections.DefaultReplication = 2
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
      expect_max_sig_exp = db_current_time.to_i + Rails.configuration.Collections.BlobSigningTTL.to_i
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
                   Rails.configuration.Collections.BlobSigningTTL)
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
    Rails.configuration.Collections.DefaultTrashLifetime = 86400 * 21 # 3 weeks
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
      assert_raises ActiveRecord::RecordInvalid do
        # Cannot create because :trashed_on_next_sweep is already trashed
        Link.create!(head_uuid: uuid,
                     tail_uuid: system_user_uuid,
                     link_class: 'whatever',
                     name: 'something')
      end

      # Bump trash_at to now + 1 minute
      Collection.where(uuid: uuid).
        update(trash_at: db_current_time + (1).minute)

      # Not considered trashed now
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

  test "empty names are exempt from name uniqueness" do
    act_as_user users(:active) do
      c1 = Collection.new(name: nil, manifest_text: '', owner_uuid: groups(:aproject).uuid)
      assert c1.save
      c2 = Collection.new(name: '', manifest_text: '', owner_uuid: groups(:aproject).uuid)
      assert c2.save
      c3 = Collection.new(name: '', manifest_text: '', owner_uuid: groups(:aproject).uuid)
      assert c3.save
      c4 = Collection.new(name: 'c4', manifest_text: '', owner_uuid: groups(:aproject).uuid)
      assert c4.save
      c5 = Collection.new(name: 'c4', manifest_text: '', owner_uuid: groups(:aproject).uuid)
      assert_raises(ActiveRecord::RecordNotUnique) do
        c5.save
      end
    end
  end

  test "create collections with managed properties" do
    Rails.configuration.Collections.ManagedProperties = ConfigLoader.to_OrderedOptions({
      'default_prop1' => {'Value' => 'prop1_value'},
      'responsible_person_uuid' => {'Function' => 'original_owner'}
    })
    # Test collection without initial properties
    act_as_user users(:active) do
      c = create_collection 'foo', Encoding::US_ASCII
      assert c.valid?
      assert_not_empty c.properties
      assert_equal 'prop1_value', c.properties['default_prop1']
      assert_equal users(:active).uuid, c.properties['responsible_person_uuid']
    end
    # Test collection with default_prop1 property already set
    act_as_user users(:active) do
      c = Collection.create(manifest_text: ". d41d8cd98f00b204e9800998ecf8427e 0:34:foo.txt\n",
                            properties: {'default_prop1' => 'custom_value'})
      assert c.valid?
      assert_not_empty c.properties
      assert_equal 'custom_value', c.properties['default_prop1']
      assert_equal users(:active).uuid, c.properties['responsible_person_uuid']
    end
    # Test collection inside a sub project
    act_as_user users(:active) do
      c = Collection.create(manifest_text: ". d41d8cd98f00b204e9800998ecf8427e 0:34:foo.txt\n",
                            owner_uuid: groups(:asubproject).uuid)
      assert c.valid?
      assert_not_empty c.properties
      assert_equal users(:active).uuid, c.properties['responsible_person_uuid']
    end
  end

  test "update collection with protected managed properties" do
    Rails.configuration.Collections.ManagedProperties = ConfigLoader.to_OrderedOptions({
      'default_prop1' => {'Value' => 'prop1_value', 'Protected' => true},
    })
    act_as_user users(:active) do
      c = create_collection 'foo', Encoding::US_ASCII
      assert c.valid?
      assert_not_empty c.properties
      assert_equal 'prop1_value', c.properties['default_prop1']
      # Add new property
      c.properties['prop2'] = 'value2'
      c.save!
      c.reload
      assert_equal 'value2', c.properties['prop2']
      # Try to change protected property's value
      c.properties['default_prop1'] = 'new_value'
      assert_raises(ArvadosModel::PermissionDeniedError) do
        c.save!
      end
      # Admins are allowed to change protected properties
      act_as_system_user do
        c.properties['default_prop1'] = 'new_value'
        c.save!
        c.reload
        assert_equal 'new_value', c.properties['default_prop1']
      end
    end
  end

  test "collection names must be displayable in a filesystem" do
    set_user_from_auth :active
    ["", "{SOLIDUS}"].each do |subst|
      Rails.configuration.Collections.ForwardSlashNameSubstitution = subst
      c = Collection.create
      [[nil, true],
       ["", true],
       [".", false],
       ["..", false],
       ["...", true],
       ["..z..", true],
       ["foo/bar", subst != ""],
       ["../..", subst != ""],
       ["/", subst != ""],
      ].each do |name, valid|
        c.name = name
        assert_equal valid, c.valid?, "#{name.inspect} should be #{valid ? "valid" : "invalid"}"
      end
    end
  end
end
