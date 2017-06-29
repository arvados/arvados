# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

require 'test_helper'
require 'salvage_collection'
require 'shellwords'

# Valid manifest_text
TEST_MANIFEST = ". 341dabea2bd78ad0d6fc3f5b926b450e+85626+Ad391622a17f61e4a254eda85d1ca751c4f368da9@55e076ce 0:85626:brca2-hg19.fa\n. d7321a918923627c972d8f8080c07d29+82570+A22e0a1d9b9bc85c848379d98bedc64238b0b1532@55e076ce 0:82570:brca1-hg19.fa\n"
TEST_MANIFEST_STRIPPED = ". 341dabea2bd78ad0d6fc3f5b926b450e+85626 0:85626:brca2-hg19.fa\n. d7321a918923627c972d8f8080c07d29+82570 0:82570:brca1-hg19.fa\n"

# This invalid manifest_text has the following flaws:
#   Missing stream name with locator in it's place
#   Invalid locators:
#     foofaafaafaabd78ad0d6fc3f5b926b450e+foo
#     bar-baabaabaabd78ad0d6fc3f5b926b450e
#     bad12345dae58ad0d6fc3f5b926b450e+
#     341dabea2bd78ad0d6fc3f5b926b450e+abc
#     341dabea2bd78ad0d6fc3f5b926abcdf
# Expectation: All these locators are preserved in salvaged_data
BAD_MANIFEST = "faafaafaabd78ad0d6fc3f5b926b450e+foo bar-baabaabaabd78ad0d6fc3f5b926b450e_bad12345dae58ad0d6fc3f5b926b450e+ 341dabea2bd78ad0d6fc3f5b926b450e+abc 341dabea2bd78ad0d6fc3f5b926abcdf 0:85626:brca2-hg19.fa\n. abcdabea2bd78ad0d6fc3f5b926b450e+1000 0:1000:brca-hg19.fa\n. d7321a918923627c972d8f8080c07d29+2000+A22e0a1d9b9bc85c848379d98bedc64238b0b1532@55e076ce 0:2000:brca1-hg19.fa\n"

class SalvageCollectionTest < ActiveSupport::TestCase
  include SalvageCollection

  setup do
    set_user_from_auth :admin
    # arv-put needs ARV env variables
    ENV['ARVADOS_API_HOST'] = 'unused_by_test'
    ENV['ARVADOS_API_TOKEN'] = 'unused_by_test'
    @backtick_mock_failure = false
  end

  teardown do
    ENV['ARVADOS_API_HOST'] = ''
    ENV['ARVADOS_API_TOKEN'] = ''
  end

  def ` cmd # mock Kernel `
    assert_equal 'arv-put', cmd.shellsplit[0]
    if @backtick_mock_failure
      # run a process so $? indicates failure
      return super 'false'
    end
    # run a process so $? indicates success
    super 'true'
    file_contents = File.open(cmd.shellsplit[-1], "r").read
    ". " +
      Digest::MD5.hexdigest(file_contents) + "+" + file_contents.length.to_s +
      " 0:" + file_contents.length.to_s + ":invalid_manifest_text.txt\n"
  end

  test "salvage test collection with valid manifest text" do
    # create a collection to test salvaging
    src_collection = Collection.new name: "test collection", manifest_text: TEST_MANIFEST
    src_collection.save!

    # salvage this collection
    salvage_collection src_collection.uuid, 'test salvage collection - see #6277, #6859'

    # verify the updated src_collection data
    updated_src_collection = Collection.find_by_uuid src_collection.uuid
    updated_name = updated_src_collection.name
    assert_equal true, updated_name.include?(src_collection.name)

    match = updated_name.match(/^test collection.*salvaged data at (.*)\)$/)
    assert_not_nil match
    assert_not_nil match[1]
    assert_empty updated_src_collection.manifest_text

    # match[1] is the uuid of the new collection created from src_collection's salvaged data
    # use this to get the new collection and verify
    new_collection = Collection.find_by_uuid match[1]
    match = new_collection.name.match(/^salvaged from (.*),.*/)
    assert_not_nil match
    assert_equal src_collection.uuid, match[1]

    # verify the new collection's manifest format
    expected_manifest = ". " + Digest::MD5.hexdigest(TEST_MANIFEST_STRIPPED) + "+" +
      TEST_MANIFEST_STRIPPED.length.to_s + " 0:" + TEST_MANIFEST_STRIPPED.length.to_s +
      ":invalid_manifest_text.txt\n. 341dabea2bd78ad0d6fc3f5b926b450e+85626 d7321a918923627c972d8f8080c07d29+82570 0:168196:salvaged_data\n"
    assert_equal expected_manifest, new_collection.manifest_text
  end

  test "salvage collection with no uuid required argument" do
    assert_raises RuntimeError do
      salvage_collection nil
    end
  end

  test "salvage collection with bogus uuid" do
    e = assert_raises RuntimeError do
      salvage_collection 'bogus-uuid'
    end
    assert_equal "No collection found for bogus-uuid.", e.message
  end

  test "salvage collection with no env ARVADOS_API_HOST" do
    e = assert_raises RuntimeError do
      ENV['ARVADOS_API_HOST'] = ''
      ENV['ARVADOS_API_TOKEN'] = ''
      salvage_collection collections('user_agreement').uuid
    end
    assert_equal "ARVADOS environment variables missing. Please set your admin user credentials as ARVADOS environment variables.", e.message
  end

  test "salvage collection with error during arv-put" do
    # try to salvage collection while mimicking error during arv-put
    @backtick_mock_failure = true
    e = assert_raises RuntimeError do
      salvage_collection collections('user_agreement').uuid
    end
    assert_match(/Error during arv-put: pid \d+ exit \d+ \(cmd was \"arv-put .*\"\)/, e.message)
  end

  # This test uses BAD_MANIFEST, which has the following flaws:
  #   Missing stream name with locator in it's place
  #   Invalid locators:
  #     foo-faafaafaabd78ad0d6fc3f5b926b450e+foo
  #     bar-baabaabaabd78ad0d6fc3f5b926b450e
  #     bad12345dae58ad0d6fc3f5b926b450e+
  #     341dabea2bd78ad0d6fc3f5b926b450e+abc
  #     341dabea2bd78ad0d6fc3f5b926abcdf
  # Expectation: All these locators are preserved in salvaged_data
  test "invalid locators preserved during salvaging" do
    locator_data = salvage_collection_locator_data BAD_MANIFEST
    assert_equal \
    ["faafaafaabd78ad0d6fc3f5b926b450e",
     "baabaabaabd78ad0d6fc3f5b926b450e",
     "bad12345dae58ad0d6fc3f5b926b450e",
     "341dabea2bd78ad0d6fc3f5b926b450e",
     "341dabea2bd78ad0d6fc3f5b926abcdf",
     "abcdabea2bd78ad0d6fc3f5b926b450e+1000",
     "d7321a918923627c972d8f8080c07d29+2000",
    ], locator_data[0]
    assert_equal 1000+2000, locator_data[1]
  end

  test "salvage a collection with invalid manifest text" do
    # create a collection to test salvaging
    src_collection = Collection.new name: "test collection", manifest_text: BAD_MANIFEST, owner_uuid: 'zzzzz-tpzed-000000000000000'
    src_collection.save!(validate: false)

    # salvage this collection
    salvage_collection src_collection.uuid, 'test salvage collection - see #6277, #6859'

    # verify the updated src_collection data
    updated_src_collection = Collection.find_by_uuid src_collection.uuid
    updated_name = updated_src_collection.name
    assert_equal true, updated_name.include?(src_collection.name)

    match = updated_name.match(/^test collection.*salvaged data at (.*)\)$/)
    assert_not_nil match
    assert_not_nil match[1]
    assert_empty updated_src_collection.manifest_text

    # match[1] is the uuid of the new collection created from src_collection's salvaged data
    # use this to get the new collection and verify
    new_collection = Collection.find_by_uuid match[1]
    match = new_collection.name.match(/^salvaged from (.*),.*/)
    assert_not_nil match
    assert_equal src_collection.uuid, match[1]
    # verify the new collection's manifest includes the bad locators
    expected_manifest = ". " + Digest::MD5.hexdigest(BAD_MANIFEST) + "+" + BAD_MANIFEST.length.to_s +
      " 0:" + BAD_MANIFEST.length.to_s + ":invalid_manifest_text.txt\n. faafaafaabd78ad0d6fc3f5b926b450e baabaabaabd78ad0d6fc3f5b926b450e bad12345dae58ad0d6fc3f5b926b450e 341dabea2bd78ad0d6fc3f5b926b450e 341dabea2bd78ad0d6fc3f5b926abcdf abcdabea2bd78ad0d6fc3f5b926b450e+1000 d7321a918923627c972d8f8080c07d29+2000 0:3000:salvaged_data\n"
    assert_equal expected_manifest, new_collection.manifest_text
  end
end
