require 'test_helper'

class CollectionTest < ActiveSupport::TestCase
  include CollectionsHelper

  test 'recognize empty blob locator' do
    ['d41d8cd98f00b204e9800998ecf8427e+0',
     'd41d8cd98f00b204e9800998ecf8427e',
     'd41d8cd98f00b204e9800998ecf8427e+0+Xyzzy'].each do |x|
      assert_equal true, Collection.is_empty_blob_locator?(x)
    end
    ['d41d8cd98f00b204e9800998ecf8427e0',
     'acbd18db4cc2f85cedef654fccc4a4d8+3',
     'acbd18db4cc2f85cedef654fccc4a4d8+0'].each do |x|
      assert_equal false, Collection.is_empty_blob_locator?(x)
    end
  end

  def get_files_tree(coll_name)
    use_token :admin
    Collection.find(api_fixture('collections')[coll_name]['uuid']).files_tree
  end

  test "easy files_tree" do
    files_in = lambda do |dirname|
      (1..3).map { |n| [dirname, "file#{n}", 0] }
    end
    assert_equal([['.', 'dir1', nil], ['./dir1', 'subdir', nil]] +
                 files_in['./dir1/subdir'] + files_in['./dir1'] +
                 [['.', 'dir2', nil]] + files_in['./dir2'] + files_in['.'],
                 get_files_tree('multilevel_collection_1'),
                 "Collection file tree was malformed")
  end

  test "files_tree with files deep in subdirectories" do
    # This test makes sure files_tree generates synthetic directory entries.
    # The manifest doesn't list directories with no files.
    assert_equal([['.', 'dir1', nil], ['./dir1', 'sub1', nil],
                  ['./dir1/sub1', 'a', 0], ['./dir1/sub1', 'b', 0],
                  ['.', 'dir2', nil], ['./dir2', 'sub2', nil],
                  ['./dir2/sub2', 'c', 0], ['./dir2/sub2', 'd', 0]],
                 get_files_tree('multilevel_collection_2'),
                 "Collection file tree was malformed")
  end

  test "portable_data_hash never editable" do
    refute(Collection.new.attribute_editable?("portable_data_hash", :ever))
  end

  test "admin can edit name" do
    use_token :admin
    assert(find_fixture(Collection, "foo_file").attribute_editable?("name"),
           "admin not allowed to edit collection name")
  end

  test "project owner can edit name" do
    use_token :active
    assert(find_fixture(Collection, "foo_collection_in_aproject")
             .attribute_editable?("name"),
           "project owner not allowed to edit collection name")
  end

  test "project admin can edit name" do
    use_token :subproject_admin
    assert(find_fixture(Collection, "baz_file_in_asubproject")
             .attribute_editable?("name"),
           "project admin not allowed to edit collection name")
  end

  test "project viewer cannot edit name" do
    use_token :project_viewer
    refute(find_fixture(Collection, "foo_collection_in_aproject")
             .attribute_editable?("name"),
           "project viewer allowed to edit collection name")
  end

  [
    ["filename.csv", true],
    ["filename.fa", true],
    ["filename.fasta", true],
    ["filename.seq", true],   # another fasta extension
    ["filename.go", true],
    ["filename.htm", true],
    ["filename.html", true],
    ["filename.json", true],
    ["filename.md", true],
    ["filename.pdf", true],
    ["filename.py", true],
    ["filename.R", true],
    ["filename.sam", true],
    ["filename.sh", true],
    ["filename.txt", true],
    ["filename.tiff", true],
    ["filename.tsv", true],
    ["filename.vcf", true],
    ["filename.xml", true],
    ["filename.xsl", true],
    ["filename.yml", true],

    ["filename.bam", false],
    ["filename", false],
  ].each do |file_name, preview_allowed|
    test "verify '#{file_name}' is allowed for preview #{preview_allowed}" do
      assert_equal preview_allowed, preview_allowed_for(file_name)
    end
  end
end
