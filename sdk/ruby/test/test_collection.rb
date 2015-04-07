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

  ### .normalize

  def test_non_posix_path_handling
    m_text = "./.. #{random_block(9)} 0:5:. 5:4:..\n"
    coll = Arv::Collection.new(m_text.dup)
    coll.normalize
    assert_equal(m_text, coll.manifest_text)
  end

  def test_escaping_through_normalization
    coll = Arv::Collection.new(MANY_ESCAPES_MANIFEST)
    coll.normalize
    # The result should simply duplicate the file spec.
    # The source file spec has an unescaped backslash in it.
    # It's OK for the Collection class to properly escape that.
    expect_text = MANY_ESCAPES_MANIFEST.sub(/ \d+:\d+:\S+/) do |file_spec|
      file_spec.gsub(/([^\\])(\\[^\\\d])/, '\1\\\\\2')
    end
    assert_equal(expect_text, coll.manifest_text)
  end

  def test_concatenation_with_locator_overlap(over_index=0)
    blocks = random_blocks(4, 2)
    blocks_s = blocks.join(" ")
    coll = Arv::Collection.new(". %s 0:8:file\n. %s 0:4:file\n" %
                               [blocks_s, blocks[over_index, 2].join(" ")])
    coll.normalize
    assert_equal(". #{blocks_s} 0:8:file #{over_index * 2}:4:file\n",
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
    coll = Arv::Collection
      .new(". %s 0:6:overlap\n. %s 0:6:overlap\n" %
           [blocks[0, 2].join(" "), blocks[1, 2].join(" ")])
    coll.normalize
    assert_equal(". #{blocks.join(' ')} 0:6:overlap 3:6:overlap\n",
                 coll.manifest_text)
  end

  def test_normalize
    block = random_block
    coll = Arv::Collection.new(". #{block} 0:0:f2 0:0:f1\n")
    coll.normalize
    assert_equal(". #{block} 0:0:f1 0:0:f2\n", coll.manifest_text)
  end

  def test_normalization_file_spans_two_whole_blocks(file_specs="0:10:f1",
                                                     num_blocks=2)
    blocks = random_blocks(num_blocks, 5)
    m_text = ". #{blocks.join(' ')} #{file_specs}\n"
    coll = Arv::Collection.new(m_text.dup)
    coll.normalize
    assert_equal(m_text, coll.manifest_text)
  end

  def test_normalization_file_fits_beginning_block
    test_normalization_file_spans_two_whole_blocks("0:7:f1")
  end

  def test_normalization_file_fits_end_block
    test_normalization_file_spans_two_whole_blocks("3:7:f1")
  end

  def test_normalization_file_spans_middle
    test_normalization_file_spans_two_whole_blocks("3:5:f1")
  end

  def test_normalization_file_spans_three_whole_blocks
    test_normalization_file_spans_two_whole_blocks("0:15:f1", 3)
  end

  def test_normalization_file_skips_bytes
    test_normalization_file_spans_two_whole_blocks("0:3:f1 5:5:f1")
  end

  def test_normalization_file_inserts_bytes
    test_normalization_file_spans_two_whole_blocks("0:3:f1 5:3:f1 3:2:f1")
  end

  def test_normalization_file_duplicates_bytes
    test_normalization_file_spans_two_whole_blocks("2:3:f1 2:3:f1", 1)
  end

  def test_normalization_dedups_locators
    blocks = random_blocks(2, 5)
    coll = Arv::Collection.new(". %s %s 1:8:f1 11:8:f1\n" %
                               [blocks.join(" "), blocks.reverse.join(" ")])
    coll.normalize
    assert_equal(". #{blocks.join(' ')} 1:8:f1 6:4:f1 0:4:f1\n",
                 coll.manifest_text)
  end

  ### .cp_r

  def test_simple_file_copy
    coll = Arv::Collection.new(SIMPLEST_MANIFEST)
    coll.cp_r("./simple.txt", "./new")
    assert_equal(SIMPLEST_MANIFEST.sub(" 0:9:", " 0:9:new 0:9:"),
                 coll.manifest_text)
  end

  def test_copy_file_into_other_stream(target="./s1/f2", basename="f2")
    coll = Arv::Collection.new(TWO_BY_TWO_MANIFEST_S)
    coll.cp_r("./f2", target)
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
    coll.cp_r("./f1", target)
    expected = "%s./s1 %s 0:5:f1 14:4:f3\n" %
      [TWO_BY_TWO_MANIFEST_A.first, TWO_BY_TWO_BLOCKS.join(" ")]
    assert_equal(expected, coll.manifest_text)
  end

  def test_implicit_copy_file_over_in_other_stream
    test_copy_file_over_in_other_stream("./s1")
  end

  def test_simple_stream_copy
    coll = Arv::Collection.new(TWO_BY_TWO_MANIFEST_S)
    coll.cp_r("./s1", "./sNew")
    new_line = TWO_BY_TWO_MANIFEST_A.last.sub("./s1 ", "./sNew ")
    assert_equal(TWO_BY_TWO_MANIFEST_S + new_line, coll.manifest_text)
  end

  def test_copy_stream_into_other_stream(target="./dir2/subdir",
                                         basename="subdir")
    coll = Arv::Collection.new(MULTILEVEL_MANIFEST)
    coll.cp_r("./dir1/subdir", target)
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
      coll.rm("./dir0/subdir/file#{file_num}")
    end
    coll.cp_r("./dir1/subdir", "./dir0")
    expected = MULTILEVEL_MANIFEST.lines
    expected[2] = expected[4].sub("./dir1/", "./dir0/")
    assert_equal(expected.join(""), coll.manifest_text)
  end

  def test_copy_stream_over_file_raises_ENOTDIR
    coll = Arv::Collection.new(TWO_BY_TWO_MANIFEST_S)
    assert_raises(Errno::ENOTDIR) do
      coll.cp_r("./s1", "./f2")
    end
  end

  def test_copy_stream_over_nonempty_stream_merges_and_overwrites
    blocks = random_blocks(3, 9)
    manifest_a =
      ["./subdir #{blocks[0]} 0:1:s1 1:2:zero\n",
       "./zdir #{blocks[1]} 0:9:zfile\n",
       "./zdir/subdir #{blocks[2]} 0:1:s2 1:2:zero\n"]
    coll = Arv::Collection.new(manifest_a.join(""))
    coll.cp_r("./subdir", "./zdir")
    manifest_a[2] = "./zdir/subdir %s %s 0:1:s1 9:1:s2 1:2:zero\n" %
      [blocks[0], blocks[2]]
    assert_equal(manifest_a.join(""), coll.manifest_text)
  end

  def test_copy_stream_into_substream(source="./dir1",
                                      target="./dir1/subdir/dir1")
    coll = Arv::Collection.new(MULTILEVEL_MANIFEST)
    coll.cp_r(source, target)
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
    coll.cp_r(".", "./root")
    src_coll = Arv::Collection.new(COLON_FILENAME_MANIFEST)
    coll.cp_r("./file:test.txt", ".", src_coll)
    got_lines = coll.manifest_text.lines
    assert_equal(2, got_lines.size)
    assert_match(/^\. \S{33,} \S{33,} 0:9:file:test\.txt 9:9:simple\.txt\n/,
                 got_lines.first)
    assert_equal(SIMPLEST_MANIFEST.sub(". ", "./root "), got_lines.last)
  end

  def test_copy_chaining
    coll = Arv::Collection.new(SIMPLEST_MANIFEST)
    coll.cp_r("./simple.txt", "./a").cp_r("./a", "./b")
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
    dst_coll.cp_r("#{src_stream}/f1", dst_stream, src_coll)
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
    dst_coll.cp_r("./s2", "./s1", src_coll)
    assert_equal(dst_text + src_text.sub("./s2 ", "./s1/s2 "),
                 dst_coll.manifest_text)
    assert_equal(src_text, src_coll.manifest_text)
  end

  def test_copy_stream_from_other_collection_to_root
    blocks, src_text, dst_text, src_coll, dst_coll =
      prep_two_collections_for_copy("./s1", ".")
    dst_coll.cp_r("./s1", ".", src_coll)
    assert_equal(dst_text + src_text, dst_coll.manifest_text)
    assert_equal(src_text, src_coll.manifest_text)
  end

  def test_copy_stream_contents
    coll = Arv::Collection.new(MULTILEVEL_MANIFEST)
    coll.cp_r("./dir0/subdir/", "./dir1/subdir")
    expect_lines = MULTILEVEL_MANIFEST.lines
    expect_lines[4] = expect_lines[2].sub("./dir0/", "./dir1/")
    assert_equal(expect_lines.join(""), coll.manifest_text)
  end

  def test_copy_stream_contents_into_root
    coll = Arv::Collection.new(TWO_BY_TWO_MANIFEST_S)
    coll.cp_r("./s1/", ".")
    assert_equal(". %s 0:5:f1 14:4:f2 5:4:f3\n%s" %
                 [TWO_BY_TWO_BLOCKS.reverse.join(" "),
                  TWO_BY_TWO_MANIFEST_A.last],
                 coll.manifest_text)
  end

  def test_copy_root_contents_into_stream
    # This is especially fun, because we're copying a parent into its child.
    # Make sure that happens depth-first.
    coll = Arv::Collection.new(TWO_BY_TWO_MANIFEST_S)
    coll.cp_r("./", "./s1")
    assert_equal("%s./s1 %s 0:5:f1 5:4:f2 14:4:f3\n%s" %
                 [TWO_BY_TWO_MANIFEST_A.first, TWO_BY_TWO_BLOCKS.join(" "),
                  TWO_BY_TWO_MANIFEST_A.last.sub("./s1 ", "./s1/s1 ")],
                 coll.manifest_text)
  end

  def test_copy_stream_contents_across_collections
    block = random_block(8)
    src_coll = Arv::Collection.new("./s1 #{block} 0:8:f1\n")
    dst_coll = Arv::Collection.new(TWO_BY_TWO_MANIFEST_S)
    dst_coll.cp_r("./s1/", "./s1", src_coll)
    assert_equal("%s./s1 %s %s 0:8:f1 13:4:f3\n" %
                 [TWO_BY_TWO_MANIFEST_A.first, block, TWO_BY_TWO_BLOCKS.last],
                 dst_coll.manifest_text)
  end

  def test_copy_root_contents_across_collections
    block = random_block(8)
    src_coll = Arv::Collection.new(". #{block} 0:8:f1\n")
    dst_coll = Arv::Collection.new(TWO_BY_TWO_MANIFEST_S)
    dst_coll.cp_r("./", ".", src_coll)
    assert_equal(". %s %s 0:8:f1 13:4:f2\n%s" %
                 [block, TWO_BY_TWO_BLOCKS.first, TWO_BY_TWO_MANIFEST_A.last],
                 dst_coll.manifest_text)
  end

  def test_copy_empty_source_path_raises_ArgumentError(src="", dst="./s1")
    coll = Arv::Collection.new(SIMPLEST_MANIFEST)
    assert_raises(ArgumentError) do
      coll.cp_r(src, dst)
    end
  end

  def test_copy_empty_destination_path_raises_ArgumentError
    test_copy_empty_source_path_raises_ArgumentError(".", "")
  end

  ### .each_file_path

  def test_each_file_path
    coll = Arv::Collection.new(TWO_BY_TWO_MANIFEST_S)
    if block_given?
      result = yield(coll)
    else
      result = []
      coll.each_file_path { |path| result << path }
    end
    assert_equal(["./f1", "./f2", "./s1/f1", "./s1/f3"], result.sort)
  end

  def test_each_file_path_without_block
    test_each_file_path { |coll| coll.each_file_path.to_a }
  end

  def test_each_file_path_empty_collection
    assert_empty(Arv::Collection.new.each_file_path.to_a)
  end

  def test_each_file_path_after_collection_emptied
    coll = Arv::Collection.new(SIMPLEST_MANIFEST)
    coll.rm("simple.txt")
    assert_empty(coll.each_file_path.to_a)
  end

  def test_each_file_path_deduplicates_manifest_listings
    coll = Arv::Collection.new(MULTIBLOCK_FILE_MANIFEST)
    assert_equal(["./repfile", "./s1/repfile", "./s1/uniqfile",
                  "./uniqfile", "./uniqfile2"],
                 coll.each_file_path.to_a.sort)
  end

  ### .exist?

  def test_exist(test_method=:assert, path="f2")
    coll = Arv::Collection.new(TWO_BY_TWO_MANIFEST_S)
    send(test_method, coll.exist?(path))
  end

  def test_file_not_exist
    test_exist(:refute, "f3")
  end

  def test_stream_exist
    test_exist(:assert, "s1")
  end

  def test_file_inside_stream_exist
    test_exist(:assert, "s1/f1")
  end

  def test_path_inside_stream_not_exist
    test_exist(:refute, "s1/f2")
  end

  def test_path_under_file_not_exist
    test_exist(:refute, "f2/nonexistent")
  end

  def test_deep_substreams_not_exist
    test_exist(:refute, "a/b/c/d/e/f/g")
  end

  ### .rename

  def test_simple_file_rename
    coll = Arv::Collection.new(SIMPLEST_MANIFEST)
    coll.rename("./simple.txt", "./new")
    assert_equal(SIMPLEST_MANIFEST.sub(":simple.txt", ":new"),
                 coll.manifest_text)
  end

  def test_rename_file_into_other_stream(target="./s1/f2", basename="f2")
    coll = Arv::Collection.new(TWO_BY_TWO_MANIFEST_S)
    coll.rename("./f2", target)
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
    coll.rename("./f1", target)
    expected = ". %s 5:4:f2\n./s1 %s 0:5:f1 14:4:f3\n" %
      [TWO_BY_TWO_BLOCKS.first, TWO_BY_TWO_BLOCKS.join(" ")]
    assert_equal(expected, coll.manifest_text)
  end

  def test_implicit_rename_file_over_in_other_stream
    test_rename_file_over_in_other_stream("./s1")
  end

  def test_simple_stream_rename
    coll = Arv::Collection.new(TWO_BY_TWO_MANIFEST_S)
    coll.rename("./s1", "./newS")
    assert_equal(TWO_BY_TWO_MANIFEST_S.sub("\n./s1 ", "\n./newS "),
                 coll.manifest_text)
  end

  def test_rename_stream_into_other_stream(target="./dir2/subdir",
                                           basename="subdir")
    coll = Arv::Collection.new(MULTILEVEL_MANIFEST)
    coll.rename("./dir1/subdir", target)
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
      coll.rm("./dir0/subdir/file#{file_num}")
    end
    coll.rename("./dir1/subdir", "./dir0")
    expected = MULTILEVEL_MANIFEST.lines
    expected[2] = expected.delete_at(4).sub("./dir1/", "./dir0/")
    assert_equal(expected.sort.join(""), coll.manifest_text)
  end

  def test_rename_stream_over_file_raises_ENOTDIR
    coll = Arv::Collection.new(TWO_BY_TWO_MANIFEST_S)
    assert_raises(Errno::ENOTDIR) do
      coll.rename("./s1", "./f2")
    end
  end

  def test_rename_stream_over_nonempty_stream_raises_ENOTEMPTY
    coll = Arv::Collection.new(MULTILEVEL_MANIFEST)
    assert_raises(Errno::ENOTEMPTY) do
      coll.rename("./dir1/subdir", "./dir0")
    end
  end

  def test_rename_stream_into_substream(source="./dir1",
                                        target="./dir1/subdir/dir1")
    coll = Arv::Collection.new(MULTILEVEL_MANIFEST)
    coll.rename(source, target)
    assert_equal(MULTILEVEL_MANIFEST.gsub(/^#{Regexp.escape(source)}([\/ ])/m,
                                          "#{target}\\1"),
                 coll.manifest_text)
  end

  def test_rename_root
    test_rename_stream_into_substream(".", "./root")
  end

  def test_adding_to_root_after_rename
    coll = Arv::Collection.new(SIMPLEST_MANIFEST)
    coll.rename(".", "./root")
    src_coll = Arv::Collection.new(SIMPLEST_MANIFEST)
    coll.cp_r("./simple.txt", ".", src_coll)
    assert_equal(SIMPLEST_MANIFEST + SIMPLEST_MANIFEST.sub(". ", "./root "),
                 coll.manifest_text)
  end

  def test_rename_chaining
    coll = Arv::Collection.new(SIMPLEST_MANIFEST)
    coll.rename("./simple.txt", "./x").rename("./x", "./simple.txt")
    assert_equal(SIMPLEST_MANIFEST, coll.manifest_text)
  end

  ### .rm

  def test_simple_remove
    coll = Arv::Collection.new(TWO_BY_TWO_MANIFEST_S.dup)
    coll.rm("./f2")
    assert_equal(TWO_BY_TWO_MANIFEST_S.sub(" 5:4:f2", ""), coll.manifest_text)
  end

  def empty_stream_and_assert(expect_index=0)
    coll = Arv::Collection.new(TWO_BY_TWO_MANIFEST_S)
    yield coll
    assert_equal(TWO_BY_TWO_MANIFEST_A[expect_index], coll.manifest_text)
  end

  def test_remove_all_files_in_substream
    empty_stream_and_assert do |coll|
      coll.rm("./s1/f1")
      coll.rm("./s1/f3")
    end
  end

  def test_remove_all_files_in_root_stream
    empty_stream_and_assert(1) do |coll|
      coll.rm("./f1")
      coll.rm("./f2")
    end
  end

  def test_chaining_removes
    empty_stream_and_assert do |coll|
      coll.rm("./s1/f1").rm("./s1/f3")
    end
  end

  def test_remove_last_file
    coll = Arv::Collection.new(SIMPLEST_MANIFEST)
    coll.rm("./simple.txt")
    assert_equal("", coll.manifest_text)
  end

  def test_remove_nonexistent_file_raises_ENOENT(path="./NoSuchFile",
                                                 method=:rm)
    coll = Arv::Collection.new(SIMPLEST_MANIFEST)
    assert_raises(Errno::ENOENT) do
      coll.send(method, path)
    end
  end

  def test_remove_from_nonexistent_stream_raises_ENOENT
    test_remove_nonexistent_file_raises_ENOENT("./NoSuchStream/simple.txt")
  end

  def test_remove_stream_raises_EISDIR(path="./s1")
    coll = Arv::Collection.new(TWO_BY_TWO_MANIFEST_S)
    assert_raises(Errno::EISDIR) do
      coll.rm(path)
    end
  end

  def test_remove_root_raises_EISDIR
    test_remove_stream_raises_EISDIR(".")
  end

  def test_remove_empty_string_raises_ArgumentError
    coll = Arv::Collection.new(SIMPLEST_MANIFEST)
    assert_raises(ArgumentError) do
      coll.rm("")
    end
  end

  ### rm_r

  def test_recursive_remove
    empty_stream_and_assert do |coll|
      coll.rm_r("./s1")
    end
  end

  def test_recursive_remove_on_files
    empty_stream_and_assert do |coll|
      coll.rm_r("./s1/f1")
      coll.rm_r("./s1/f3")
    end
  end

  def test_recursive_remove_root
    coll = Arv::Collection.new(MULTILEVEL_MANIFEST)
    coll.rm_r(".")
    assert_equal("", coll.manifest_text)
  end

  def test_rm_r_nonexistent_file_raises_ENOENT(path="./NoSuchFile")
    test_remove_nonexistent_file_raises_ENOENT("./NoSuchFile", :rm_r)
  end

  def test_rm_r_from_nonexistent_stream_raises_ENOENT
    test_remove_nonexistent_file_raises_ENOENT("./NoSuchStream/file", :rm_r)
  end

  def test_rm_r_empty_string_raises_ArgumentError
    coll = Arv::Collection.new(SIMPLEST_MANIFEST)
    assert_raises(ArgumentError) do
      coll.rm_r("")
    end
  end

  ### .modified?

  def test_new_collection_unmodified(*args)
    coll = Arv::Collection.new(*args)
    yield coll if block_given?
    refute(coll.modified?)
  end

  def test_collection_unmodified_after_instantiation
    test_new_collection_unmodified(SIMPLEST_MANIFEST)
  end

  def test_collection_unmodified_after_mark
    test_new_collection_unmodified(SIMPLEST_MANIFEST) do |coll|
      coll.cp_r("./simple.txt", "./copy")
      coll.unmodified
    end
  end

  def check_collection_modified
    coll = Arv::Collection.new(SIMPLEST_MANIFEST)
    yield coll
    assert(coll.modified?)
  end

  def test_collection_modified_after_copy
    check_collection_modified do |coll|
      coll.cp_r("./simple.txt", "./copy")
    end
  end

  def test_collection_modified_after_remove
    check_collection_modified do |coll|
      coll.rm("./simple.txt")
    end
  end

  def test_collection_modified_after_rename
    check_collection_modified do |coll|
      coll.rename("./simple.txt", "./newname")
    end
  end
end
