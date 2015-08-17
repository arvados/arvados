require 'test_helper'
require 'salvage_collection'

TEST_MANIFEST = ". 341dabea2bd78ad0d6fc3f5b926b450e+85626+Ad391622a17f61e4a254eda85d1ca751c4f368da9@55e076ce 0:85626:brca2-hg19.fa\n. d7321a918923627c972d8f8080c07d29+82570+A22e0a1d9b9bc85c848379d98bedc64238b0b1532@55e076ce 0:82570:brca1-hg19.fa\n"

module Kernel
  def `(cmd)    # override kernel ` method
    if cmd.include? 'arv-put'
      file_contents = file = File.new(cmd.split[-1], "r").gets

      # simulate arv-put error when it is 'user_agreement'
      if file_contents.include? 'GNU_General_Public_License'
        return ''
      else
        ". " +
        Digest::MD5.hexdigest(TEST_MANIFEST) +
        " 0:" + TEST_MANIFEST.length.to_s + ":invalid_manifest_text.txt\n"
      end
    end
  end

  def exit code
    raise "Exit code #{code}" if code == 200
  end
end

class SalvageCollectionTest < ActiveSupport::TestCase
  include SalvageCollection

  setup do
    set_user_from_auth :admin
    # arv-put needs ARV env variables
    ENV['ARVADOS_API_HOST'] = 'unused_by_test'
    ENV['ARVADOS_API_TOKEN'] = 'unused_by_test'
  end

  test "salvage test collection" do
    # create a collection to test salvaging
    src_collection = Collection.new name: "test collection", manifest_text: TEST_MANIFEST
    src_collection.save!

    # salvage this collection
    SalvageCollection.salvage_collection src_collection.uuid, 'test salvage collection - see #6277, #6859'

    # verify the updated src_collection data
    updated_src_collection = Collection.find_by_uuid src_collection.uuid
    updated_name = updated_src_collection.name
    assert_equal true, updated_name.include?(src_collection.name)

    match = updated_name.match /^test collection.*salvaged data at (.*)\)$/
    assert_not_nil match
    assert_not_nil match[1]
    assert_empty updated_src_collection.manifest_text

    # match[1] is the uuid of the new collection created from src_collection's salvaged data
    # use this to get the new collection and verify
    new_collection = Collection.find_by_uuid match[1]
    match = new_collection.name.match /^salvaged from (.*),.*/
    assert_not_nil match
    assert_equal src_collection.uuid, match[1]

    # verify the new collection's manifest format
    match = new_collection.manifest_text.match /^. (.*) (.*):invalid_manifest_text.txt\n. (.*) (.*):salvaged_data/
    assert_not_nil match
  end

  test "salvage collection with no uuid required argument" do
    status = SalvageCollection.salvage_collection nil
    assert_equal false, status
  end

  test "salvage collection with bogus uuid" do
    status = SalvageCollection.salvage_collection 'bogus-uuid'
    assert_equal false, status
  end

  test "salvage collection with no env ARVADOS_API_HOST" do
    exited = false
    begin
      ENV['ARVADOS_API_HOST'] = ''
      ENV['ARVADOS_API_TOKEN'] = ''
      SalvageCollection.salvage_collection collections('user_agreement').uuid
    rescue => e
      assert_equal "Exit code 200", e.message
      exited = true
    end
    assert_equal true, exited
  end

  test "salvage collection with during arv-put" do
    # try to salvage collection while mimicking error during arv-put
    status = SalvageCollection.salvage_collection collections('user_agreement').uuid
    assert_equal false, status
  end
end
