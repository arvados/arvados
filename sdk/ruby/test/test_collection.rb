require "arvados/collection"
require "minitest/autorun"
require "sdk_fixtures"

class CollectionTest < Minitest::Test
  include SDKFixtures

  TWO_BY_TWO_BLOCKS = SDKFixtures.random_blocks(2, 9)
  TWO_BY_TWO_MANIFEST_A =
    [". #{TWO_BY_TWO_BLOCKS.first} 0:5:f1 5:4:f2\n",
     "./s1 #{TWO_BY_TWO_BLOCKS.last} 0:5:f1 5:4:f3\n"]
  TWO_BY_TWO_MANIFEST_S = TWO_BY_TWO_MANIFEST_A.join("")

  ### .new

  def test_empty_construction
    coll = Arv::Collection.new
    assert_equal("", coll.manifest_text)
  end

  def test_successful_construction
    [:SIMPLEST_MANIFEST, :MULTIBLOCK_FILE_MANIFEST, :MULTILEVEL_MANIFEST].
        each do |manifest_name|
      manifest_text = SDKFixtures.const_get(manifest_name)
      coll = Arv::Collection.new(manifest_text)
      assert_equal(manifest_text, coll.manifest_text,
                   "did not get same manifest back out from #{manifest_name}")
    end
  end

  def test_non_manifest_construction_error
    ["word", ". abc def", ". #{random_block} 0:", ". / !"].each do |m_text|
      assert_raises(ArgumentError,
                    "built collection from manifest #{m_text.inspect}") do
        Arv::Collection.new(m_text)
      end
    end
  end

  def test_file_directory_conflict_construction_error
    assert_raises(ArgumentError) do
      Arv::Collection.new(NAME_CONFLICT_MANIFEST)
    end
  end

  def test_no_implicit_normalization
    coll = Arv::Collection.new(NONNORMALIZED_MANIFEST)
    assert_equal(NONNORMALIZED_MANIFEST, coll.manifest_text)
  end

  def test_no_implicit_normalization_from_first_import
    coll = Arv::Collection.new
    coll.import_manifest!(NONNORMALIZED_MANIFEST)
    assert_equal(NONNORMALIZED_MANIFEST, coll.manifest_text)
  end

  ### .import_manifest!

  def test_non_posix_path_handling
    block = random_block(9)
    coll = Arv::Collection.new("./.. #{block} 0:5:.\n")
    coll.import_manifest!("./.. #{block} 5:4:..\n")
    assert_equal("./.. #{block} 0:5:. 5:4:..\n", coll.manifest_text)
  end

  def test_escaping_through_normalization
    coll = Arv::Collection.new(MANY_ESCAPES_MANIFEST)
    coll.import_manifest!(MANY_ESCAPES_MANIFEST)
    # The result should simply duplicate the file spec.
    # The source file spec has an unescaped backslash in it.
    # It's OK for the Collection class to properly escape that.
    expect_text = MANY_ESCAPES_MANIFEST.sub(/ \d+:\d+:\S+/) do |file_spec|
      file_spec.gsub(/([^\\])(\\[^\\\d])/, '\1\\\\\2') * 2
    end
    assert_equal(expect_text, coll.manifest_text)
  end

  def test_concatenation_from_multiple_imports(file_name="file.txt",
                                               out_name=nil)
    out_name ||= file_name
    blocks = random_blocks(2, 9)
    coll = Arv::Collection.new
    blocks.each do |block|
      coll.import_manifest!(". #{block} 1:8:#{file_name}\n")
    end
    assert_equal(". #{blocks.join(' ')} 1:8:#{out_name} 10:8:#{out_name}\n",
                 coll.manifest_text)
  end

  def test_concatenation_from_multiple_escaped_imports
    test_concatenation_from_multiple_imports('a\040\141.txt', 'a\040a.txt')
  end

  def test_concatenation_with_locator_overlap(over_index=0)
    blocks = random_blocks(4, 2)
    coll = Arv::Collection.new(". #{blocks.join(' ')} 0:8:file\n")
    coll.import_manifest!(". #{blocks[over_index, 2].join(' ')} 0:4:file\n")
    assert_equal(". #{blocks.join(' ')} 0:8:file #{over_index * 2}:4:file\n",
                 coll.manifest_text)
  end

  def test_concatenation_with_middle_locator_overlap
    test_concatenation_with_locator_overlap(1)
  end

  def test_concatenation_with_end_locator_overlap
    test_concatenation_with_locator_overlap(2)
  end

  def test_concatenation_with_partial_locator_overlap
    blocks = random_blocks(3, 3)
    coll = Arv::Collection.new(". #{blocks[0, 2].join(' ')} 0:6:overlap\n")
    coll.import_manifest!(". #{blocks[1, 2].join(' ')} 0:6:overlap\n")
    assert_equal(". #{blocks.join(' ')} 0:6:overlap 3:6:overlap\n",
                 coll.manifest_text)
  end

  ### .normalize!

  def test_normalize
    block = random_block
    coll = Arv::Collection.new(". #{block} 0:0:f2 0:0:f1\n")
    coll.normalize!
    assert_equal(". #{block} 0:0:f1 0:0:f2\n", coll.manifest_text)
  end

  ### .copy!

  def test_simple_file_copy
    coll = Arv::Collection.new(SIMPLEST_MANIFEST)
    coll.copy!("./simple.txt", "./new")
    assert_equal(SIMPLEST_MANIFEST.sub(" 0:9:", " 0:9:new 0:9:"),
                 coll.manifest_text)
  end

  def test_copy_file_into_other_stream(target="./s1/f2", basename="f2")
    coll = Arv::Collection.new(TWO_BY_TWO_MANIFEST_S)
    coll.copy!("./f2", target)
    expected = "%s./s1 %s 0:5:f1 14:4:%s 5:4:f3\n" %
      [TWO_BY_TWO_MANIFEST_A.first,
       TWO_BY_TWO_BLOCKS.reverse.join(" "), basename]
    assert_equal(expected, coll.manifest_text)
  end

  def test_implicit_copy_file_into_other_stream
    test_copy_file_into_other_stream("./s1")
  end

  def test_copy_file_into_other_stream_with_new_name
    test_copy_file_into_other_stream("./s1/f2a", "f2a")
  end

  def test_copy_file_over_in_other_stream(target="./s1/f1")
    coll = Arv::Collection.new(TWO_BY_TWO_MANIFEST_S)
    coll.copy!("./f1", target)
    expected = "%s./s1 %s 0:5:f1 14:4:f3\n" %
      [TWO_BY_TWO_MANIFEST_A.first, TWO_BY_TWO_BLOCKS.join(" ")]
    assert_equal(expected, coll.manifest_text)
  end

  def test_implicit_copy_file_over_in_other_stream
    test_copy_file_over_in_other_stream("./s1")
  end

  def test_simple_stream_copy
    coll = Arv::Collection.new(TWO_BY_TWO_MANIFEST_S)
    coll.copy!("./s1", "./sNew")
    new_line = TWO_BY_TWO_MANIFEST_A.last.sub("./s1 ", "./sNew ")
    assert_equal(TWO_BY_TWO_MANIFEST_S + new_line, coll.manifest_text)
  end

  def test_copy_stream_into_other_stream(target="./dir2/subdir",
                                         basename="subdir")
    coll = Arv::Collection.new(MULTILEVEL_MANIFEST)
    coll.copy!("./dir1/subdir", target)
    new_line = MULTILEVEL_MANIFEST.lines[4].sub("./dir1/subdir ",
                                                "./dir2/#{basename} ")
    assert_equal(MULTILEVEL_MANIFEST + new_line, coll.manifest_text)
  end

  def test_implicit_copy_stream_into_other_stream
    test_copy_stream_into_other_stream("./dir2")
  end

  def test_copy_stream_into_other_stream_with_new_name
    test_copy_stream_into_other_stream("./dir2/newsub", "newsub")
  end

  def test_copy_stream_over_empty_stream
    coll = Arv::Collection.new(MULTILEVEL_MANIFEST)
    (1..3).each do |file_num|
      coll.remove!("./dir0/subdir/file#{file_num}")
    end
    coll.copy!("./dir1/subdir", "./dir0")
    expected = MULTILEVEL_MANIFEST.lines
    expected[2] = expected[4].sub("./dir1/", "./dir0/")
    assert_equal(expected.join(""), coll.manifest_text)
  end

  def test_copy_stream_over_file_raises_ENOTDIR
    coll = Arv::Collection.new(TWO_BY_TWO_MANIFEST_S)
    assert_raises(Errno::ENOTDIR) do
      coll.copy!("./s1", "./f2")
    end
  end

  def test_copy_stream_over_nonempty_stream_merges_and_overwrites
    blocks = random_blocks(3, 9)
    manifest_a =
      ["./subdir #{blocks[0]} 0:1:s1 1:2:zero\n",
       "./zdir #{blocks[1]} 0:9:zfile\n",
       "./zdir/subdir #{blocks[2]} 0:1:s2 1:2:zero\n"]
    coll = Arv::Collection.new(manifest_a.join(""))
    coll.copy!("./subdir", "./zdir")
    manifest_a[2] = "./zdir/subdir %s %s 0:1:s1 9:1:s2 1:2:zero\n" %
      [blocks[0], blocks[2]]
    assert_equal(manifest_a.join(""), coll.manifest_text)
  end

  def test_copy_stream_into_substream(source="./dir1",
                                      target="./dir1/subdir/dir1")
    coll = Arv::Collection.new(MULTILEVEL_MANIFEST)
    coll.copy!(source, target)
    expected = MULTILEVEL_MANIFEST.lines.flat_map do |line|
      [line, line.gsub(/^#{Regexp.escape(source)}([\/ ])/, "#{target}\\1")].uniq
    end
    assert_equal(expected.sort.join(""), coll.manifest_text)
  end

  def test_copy_root
    test_copy_stream_into_substream(".", "./root")
  end

  def test_adding_to_root_after_copy
    coll = Arv::Collection.new(SIMPLEST_MANIFEST)
    coll.copy!(".", "./root")
    coll.import_manifest!(COLON_FILENAME_MANIFEST)
    got_lines = coll.manifest_text.lines
    assert_equal(2, got_lines.size)
    assert_match(/^\. \S{33,} \S{33,} 0:9:file:test\.txt 9:9:simple\.txt\n/,
                 got_lines.first)
    assert_equal(SIMPLEST_MANIFEST.sub(". ", "./root "), got_lines.last)
  end

  def test_copy_chaining
    coll = Arv::Collection.new(SIMPLEST_MANIFEST)
    coll.copy!("./simple.txt", "./a").copy!("./a", "./b")
    assert_equal(SIMPLEST_MANIFEST.sub(" 0:9:", " 0:9:a 0:9:b 0:9:"),
                 coll.manifest_text)
  end

  def prep_two_collections_for_copy(src_stream, dst_stream)
    blocks = random_blocks(2, 8)
    src_text = "#{src_stream} #{blocks.first} 0:8:f1\n"
    dst_text = "#{dst_stream} #{blocks.last} 0:8:f2\n"
    return [blocks, src_text, dst_text,
            Arv::Collection.new(src_text.dup),
            Arv::Collection.new(dst_text.dup)]
  end

  def test_copy_file_from_other_collection(src_stream=".", dst_stream="./s1")
    blocks, src_text, dst_text, src_coll, dst_coll =
      prep_two_collections_for_copy(src_stream, dst_stream)
    dst_coll.copy!("#{src_stream}/f1", dst_stream, src_coll)
    assert_equal("#{dst_stream} #{blocks.join(' ')} 0:8:f1 8:8:f2\n",
                 dst_coll.manifest_text)
    assert_equal(src_text, src_coll.manifest_text)
  end

  def test_copy_file_from_other_collection_to_root
    test_copy_file_from_other_collection("./s1", ".")
  end

  def test_copy_stream_from_other_collection
    blocks, src_text, dst_text, src_coll, dst_coll =
      prep_two_collections_for_copy("./s2", "./s1")
    dst_coll.copy!("./s2", "./s1", src_coll)
    assert_equal(dst_text + src_text.sub("./s2 ", "./s1/s2 "),
                 dst_coll.manifest_text)
    assert_equal(src_text, src_coll.manifest_text)
  end

  def test_copy_stream_from_other_collection_to_root
    blocks, src_text, dst_text, src_coll, dst_coll =
      prep_two_collections_for_copy("./s1", ".")
    dst_coll.copy!("./s1", ".", src_coll)
    assert_equal(dst_text + src_text, dst_coll.manifest_text)
    assert_equal(src_text, src_coll.manifest_text)
  end

  def test_copy_empty_source_path_raises_ArgumentError(src="", dst="./s1")
    coll = Arv::Collection.new(SIMPLEST_MANIFEST)
    assert_raises(ArgumentError) do
      coll.copy!(src, dst)
    end
  end

  def test_copy_empty_destination_path_raises_ArgumentError
    test_copy_empty_source_path_raises_ArgumentError(".", "")
  end

  ### .rename!

  def test_simple_file_rename
    coll = Arv::Collection.new(SIMPLEST_MANIFEST)
    coll.rename!("./simple.txt", "./new")
    assert_equal(SIMPLEST_MANIFEST.sub(":simple.txt", ":new"),
                 coll.manifest_text)
  end

  def test_rename_file_into_other_stream(target="./s1/f2", basename="f2")
    coll = Arv::Collection.new(TWO_BY_TWO_MANIFEST_S)
    coll.rename!("./f2", target)
    expected = ". %s 0:5:f1\n./s1 %s 0:5:f1 14:4:%s 5:4:f3\n" %
      [TWO_BY_TWO_BLOCKS.first,
       TWO_BY_TWO_BLOCKS.reverse.join(" "), basename]
    assert_equal(expected, coll.manifest_text)
  end

  def test_implicit_rename_file_into_other_stream
    test_rename_file_into_other_stream("./s1")
  end

  def test_rename_file_into_other_stream_with_new_name
    test_rename_file_into_other_stream("./s1/f2a", "f2a")
  end

  def test_rename_file_over_in_other_stream(target="./s1/f1")
    coll = Arv::Collection.new(TWO_BY_TWO_MANIFEST_S)
    coll.rename!("./f1", target)
    expected = ". %s 5:4:f2\n./s1 %s 0:5:f1 14:4:f3\n" %
      [TWO_BY_TWO_BLOCKS.first, TWO_BY_TWO_BLOCKS.join(" ")]
    assert_equal(expected, coll.manifest_text)
  end

  def test_implicit_rename_file_over_in_other_stream
    test_rename_file_over_in_other_stream("./s1")
  end

  def test_simple_stream_rename
    coll = Arv::Collection.new(TWO_BY_TWO_MANIFEST_S)
    coll.rename!("./s1", "./newS")
    assert_equal(TWO_BY_TWO_MANIFEST_S.sub("\n./s1 ", "\n./newS "),
                 coll.manifest_text)
  end

  def test_rename_stream_into_other_stream(target="./dir2/subdir",
                                           basename="subdir")
    coll = Arv::Collection.new(MULTILEVEL_MANIFEST)
    coll.rename!("./dir1/subdir", target)
    expected = MULTILEVEL_MANIFEST.lines
    replaced_line = expected.delete_at(4)
    expected << replaced_line.sub("./dir1/subdir ", "./dir2/#{basename} ")
    assert_equal(expected.join(""), coll.manifest_text)
  end

  def test_implicit_rename_stream_into_other_stream
    test_rename_stream_into_other_stream("./dir2")
  end

  def test_rename_stream_into_other_stream_with_new_name
    test_rename_stream_into_other_stream("./dir2/newsub", "newsub")
  end

  def test_rename_stream_over_empty_stream
    coll = Arv::Collection.new(MULTILEVEL_MANIFEST)
    (1..3).each do |file_num|
      coll.remove!("./dir0/subdir/file#{file_num}")
    end
    coll.rename!("./dir1/subdir", "./dir0")
    expected = MULTILEVEL_MANIFEST.lines
    expected[2] = expected.delete_at(4).sub("./dir1/", "./dir0/")
    assert_equal(expected.sort.join(""), coll.manifest_text)
  end

  def test_rename_stream_over_file_raises_ENOTDIR
    coll = Arv::Collection.new(TWO_BY_TWO_MANIFEST_S)
    assert_raises(Errno::ENOTDIR) do
      coll.rename!("./s1", "./f2")
    end
  end

  def test_rename_stream_over_nonempty_stream_raises_ENOTEMPTY
    coll = Arv::Collection.new(MULTILEVEL_MANIFEST)
    assert_raises(Errno::ENOTEMPTY) do
      coll.rename!("./dir1/subdir", "./dir0")
    end
  end

  def test_rename_stream_into_substream(source="./dir1",
                                        target="./dir1/subdir/dir1")
    coll = Arv::Collection.new(MULTILEVEL_MANIFEST)
    coll.rename!(source, target)
    assert_equal(MULTILEVEL_MANIFEST.gsub(/^#{Regexp.escape(source)}([\/ ])/m,
                                          "#{target}\\1"),
                 coll.manifest_text)
  end

  def test_rename_root
    test_rename_stream_into_substream(".", "./root")
  end

  def test_adding_to_root_after_rename
    coll = Arv::Collection.new(SIMPLEST_MANIFEST)
    coll.rename!(".", "./root")
    coll.import_manifest!(SIMPLEST_MANIFEST)
    assert_equal(SIMPLEST_MANIFEST + SIMPLEST_MANIFEST.sub(". ", "./root "),
                 coll.manifest_text)
  end

  def test_rename_chaining
    coll = Arv::Collection.new(SIMPLEST_MANIFEST)
    coll.rename!("./simple.txt", "./x").rename!("./x", "./simple.txt")
    assert_equal(SIMPLEST_MANIFEST, coll.manifest_text)
  end

  ### .remove!

  def test_simple_remove
    coll = Arv::Collection.new(TWO_BY_TWO_MANIFEST_S.dup)
    coll.remove!("./f2")
    assert_equal(TWO_BY_TWO_MANIFEST_S.sub(" 5:4:f2", ""), coll.manifest_text)
  end

  def empty_stream_and_assert(expect_index=0)
    coll = Arv::Collection.new(TWO_BY_TWO_MANIFEST_S)
    yield coll
    assert_equal(TWO_BY_TWO_MANIFEST_A[expect_index], coll.manifest_text)
  end

  def test_remove_all_files_in_substream
    empty_stream_and_assert do |coll|
      coll.remove!("./s1/f1")
      coll.remove!("./s1/f3")
    end
  end

  def test_remove_all_files_in_root_stream
    empty_stream_and_assert(1) do |coll|
      coll.remove!("./f1")
      coll.remove!("./f2")
    end
  end

  def test_remove_empty_stream
    empty_stream_and_assert do |coll|
      coll.remove!("./s1/f1")
      coll.remove!("./s1/f3")
      coll.remove!("./s1")
    end
  end

  def test_recursive_remove
    empty_stream_and_assert do |coll|
      coll.remove!("./s1", recursive: true)
    end
  end

  def test_recursive_remove_on_files
    empty_stream_and_assert do |coll|
      coll.remove!("./s1/f1", recursive: true)
      coll.remove!("./s1/f3", recursive: true)
    end
  end

  def test_chaining_removes
    empty_stream_and_assert do |coll|
      coll.remove!("./s1/f1").remove!("./s1/f3")
    end
  end

  def test_remove_last_file
    coll = Arv::Collection.new(SIMPLEST_MANIFEST)
    coll.remove!("./simple.txt")
    assert_equal("", coll.manifest_text)
  end

  def test_remove_root_stream
    coll = Arv::Collection.new(MULTILEVEL_MANIFEST)
    coll.remove!(".", recursive: true)
    assert_equal("", coll.manifest_text)
  end

  def test_remove_nonexistent_file_raises_ENOENT(path="./NoSuchFile")
    coll = Arv::Collection.new(SIMPLEST_MANIFEST)
    assert_raises(Errno::ENOENT) do
      coll.remove!(path)
    end
  end

  def test_remove_from_nonexistent_stream_raises_ENOENT
    test_remove_nonexistent_file_raises_ENOENT("./NoSuchStream/simple.txt")
  end

  def test_remove_nonempty_stream_raises_ENOTEMPTY
    coll = Arv::Collection.new(MULTILEVEL_MANIFEST)
    assert_raises(Errno::ENOTEMPTY) do
      coll.remove!("./dir1/subdir")
    end
  end

  def test_remove_empty_string_raises_ArgumentError
    coll = Arv::Collection.new(SIMPLEST_MANIFEST)
    assert_raises(ArgumentError) do
      coll.remove!("")
    end
  end
end
