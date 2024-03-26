# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: Apache-2.0

require "arvados/keep"
require "minitest/autorun"
require "sdk_fixtures"

class ManifestTest < Minitest::Test
  include SDKFixtures

  def check_stream(stream, exp_name, exp_blocks, exp_files)
    assert_equal(exp_name, stream.first)
    assert_equal(exp_blocks, stream[1].map(&:to_s))
    assert_equal(exp_files, stream.last)
  end

  def test_simple_each_line_array
    manifest = Keep::Manifest.new(SIMPLEST_MANIFEST)
    stream_name, block_s, file = SIMPLEST_MANIFEST.strip.split
    stream_a = manifest.each_line.to_a
    assert_equal(1, stream_a.size, "wrong number of streams")
    check_stream(stream_a.first, stream_name, [block_s], [file])
  end

  def test_simple_each_line_block
    manifest = Keep::Manifest.new(SIMPLEST_MANIFEST)
    result = []
    manifest.each_line do |stream, blocks, files|
      result << files
    end
    assert_equal([[SIMPLEST_MANIFEST.split.last]], result,
                 "wrong result from each_line block")
  end

  def test_multilevel_each_line
    manifest = Keep::Manifest.new(MULTILEVEL_MANIFEST)
    seen = []
    manifest.each_line do |stream, blocks, files|
      refute(seen.include?(stream),
             "each_line already yielded stream #{stream}")
      seen << stream
      assert_equal(3, files.size, "wrong file count for stream #{stream}")
    end
    assert_equal(MULTILEVEL_MANIFEST.count("\n"), seen.size,
                 "wrong number of streams")
  end

  def test_empty_each_line
    assert_empty(Keep::Manifest.new("").each_line.to_a)
  end

  def test_empty_each_file_spec
    assert_empty(Keep::Manifest.new("").each_file_spec.to_a)
  end

  def test_empty_files
    assert_empty(Keep::Manifest.new("").files)
  end

  def test_empty_files_count
    assert_equal(0, Keep::Manifest.new("").files_count)
  end

  def test_empty_dir_files_count
    assert_equal(0,
      Keep::Manifest.new("./empty_dir d41d8cd98f00b204e9800998ecf8427e+0 0:0:\\056\n").files_count)
  end

  def test_empty_files_size
    assert_equal(0, Keep::Manifest.new("").files_size)
  end

  def test_empty_has_file?
    refute(Keep::Manifest.new("").has_file?(""))
  end

  def test_empty_line_within_manifest
    block_s = random_block
    manifest = Keep::Manifest.
      new([". #{block_s} 0:1:file1 1:2:file2\n",
           "\n",
           ". #{block_s} 3:3:file3 6:4:file4\n"].join(""))
    streams = manifest.each_line.to_a
    assert_equal(2, streams.size)
    check_stream(streams[0], ".", [block_s], ["0:1:file1", "1:2:file2"])
    check_stream(streams[1], ".", [block_s], ["3:3:file3", "6:4:file4"])
  end

  def test_backslash_escape_parsing
    manifest = Keep::Manifest.new(MANY_ESCAPES_MANIFEST)
    streams = manifest.each_line.to_a
    assert_equal(1, streams.size, "wrong number of streams with whitespace")
    assert_equal("./dir name", streams.first.first,
                 "wrong stream name with whitespace")
    assert_equal(["0:9:file\\name\t\\here.txt"], streams.first.last,
                 "wrong filename(s) with whitespace")
  end

  def test_simple_files
    manifest = Keep::Manifest.new(SIMPLEST_MANIFEST)
    assert_equal([[".", "simple.txt", 9]], manifest.files)
  end

  def test_multilevel_files
    manifest = Keep::Manifest.new(MULTILEVEL_MANIFEST)
    seen = Hash.new { |this, key| this[key] = [] }
    manifest.files.each do |stream, basename, size|
      refute(seen[stream].include?(basename),
             "each_file repeated #{stream}/#{basename}")
      seen[stream] << basename
      assert_equal(3, size, "wrong size for #{stream}/#{basename}")
    end
    seen.each_pair do |stream, basenames|
      assert_equal(%w(file1 file2 file3), basenames.sort,
                   "wrong file list for #{stream}")
    end
  end

  def test_files_with_colons_in_names
    manifest = Keep::Manifest.new(COLON_FILENAME_MANIFEST)
    assert_equal([[".", "file:test.txt", 9]], manifest.files)
  end

  def test_files_with_escape_sequence_in_filename
    manifest = Keep::Manifest.new(ESCAPED_FILENAME_MANIFEST)
    assert_equal([[".", "a a.txt", 9]], manifest.files)
  end

  def test_files_spanning_multiple_blocks
    manifest = Keep::Manifest.new(MULTIBLOCK_FILE_MANIFEST)
    assert_equal([[".", "repfile", 5],
                  [".", "uniqfile", 4],
                  [".", "uniqfile2", 7],
                  ["./s1", "repfile", 3],
                  ["./s1", "uniqfile", 3]],
                 manifest.files.sort)
  end

  def test_minimum_file_count_simple
    manifest = Keep::Manifest.new(SIMPLEST_MANIFEST)
    assert(manifest.minimum_file_count?(1), "real minimum file count false")
    refute(manifest.minimum_file_count?(2), "fake minimum file count true")
  end

  def test_minimum_file_count_multiblock
    manifest = Keep::Manifest.new(MULTIBLOCK_FILE_MANIFEST)
    assert(manifest.minimum_file_count?(2), "low minimum file count false")
    assert(manifest.minimum_file_count?(5), "real minimum file count false")
    refute(manifest.minimum_file_count?(6), "fake minimum file count true")
  end

  def test_exact_file_count_simple
    manifest = Keep::Manifest.new(SIMPLEST_MANIFEST)
    assert(manifest.exact_file_count?(1), "exact file count false")
    refute(manifest.exact_file_count?(0), "-1 file count true")
    refute(manifest.exact_file_count?(2), "+1 file count true")
  end

  def test_exact_file_count_multiblock
    manifest = Keep::Manifest.new(MULTIBLOCK_FILE_MANIFEST)
    assert(manifest.exact_file_count?(5), "exact file count false")
    refute(manifest.exact_file_count?(4), "-1 file count true")
    refute(manifest.exact_file_count?(6), "+1 file count true")
  end

  def test_files_size_multiblock
    assert_equal(22, Keep::Manifest.new(MULTIBLOCK_FILE_MANIFEST).files_size)
  end

  def test_files_size_with_skipped_overlapping_data
    manifest = Keep::Manifest.new(". #{random_block(9)} 3:3:f1 5:3:f2\n")
    assert_equal(6, manifest.files_size)
  end

  def test_has_file
    manifest = Keep::Manifest.new(MULTIBLOCK_FILE_MANIFEST)
    assert(manifest.has_file?("./repfile"), "one-arg repfile not found")
    assert(manifest.has_file?(".", "repfile"), "two-arg repfile not found")
    assert(manifest.has_file?("./s1/repfile"), "one-arg s1/repfile not found")
    assert(manifest.has_file?("./s1", "repfile"), "two-arg s1/repfile not found")
    refute(manifest.has_file?("./s1/uniqfile2"), "one-arg missing file found")
    refute(manifest.has_file?("./s1", "uniqfile2"), "two-arg missing file found")
    refute(manifest.has_file?("./s2/repfile"), "one-arg missing stream found")
    refute(manifest.has_file?("./s2", "repfile"), "two-arg missing stream found")
  end

  def test_has_file_with_spaces
    manifest = Keep::Manifest.new(ESCAPED_FILENAME_MANIFEST)
    assert(manifest.has_file?("./a a.txt"), "one-arg path not found")
    assert(manifest.has_file?(".", "a a.txt"), "two-arg path not found")
    refute(manifest.has_file?("a\\040\\141"), "one-arg unescaped found")
    refute(manifest.has_file?(".", "a\\040\\141"), "two-arg unescaped found")
  end

  def test_parse_all_fixtures
    fixtures('collections').each do |name, collection|
      parse_collection_manifest name, collection
    end
  end

  def test_raise_on_bogus_fixture
    assert_raises ArgumentError do
      parse_collection_manifest('bogus collection',
                                {'manifest_text' => ". zzz 0:\n"})
    end
  end

  def parse_collection_manifest name, collection
    manifest = Keep::Manifest.new(collection['manifest_text'])
    manifest.each_file_spec do |stream_name, start_pos, file_size, file_name|
      assert_kind_of String, stream_name
      assert_kind_of Integer, start_pos
      assert_kind_of Integer, file_size
      assert_kind_of String, file_name
      assert !stream_name.empty?, "empty stream_name in #{name} fixture"
      assert !file_name.empty?, "empty file_name in #{name} fixture"
    end
  end

  def test_collection_with_dirs_in_filenames
    manifest = Keep::Manifest.new(MANIFEST_WITH_DIRS_IN_FILENAMES)

    seen = Hash.new { |this, key| this[key] = [] }

    manifest.files.each do |stream, basename, size|
      refute(seen[stream].include?(basename), "each_file repeated #{stream}/#{basename}")
      assert_equal(3, size, "wrong size for #{stream}/#{basename}")
      seen[stream] << basename
    end

    assert_equal(%w(. ./dir1 ./dir1/dir2), seen.keys)

    seen.each_pair do |stream, basenames|
      assert_equal(%w(file1), basenames.sort, "wrong file list for #{stream}")
    end
  end

  def test_multilevel_collection_with_dirs_in_filenames
    manifest = Keep::Manifest.new(MULTILEVEL_MANIFEST_WITH_DIRS_IN_FILENAMES)

    seen = Hash.new { |this, key| this[key] = [] }
    expected_sizes = {'.' => 3, './dir1' => 6, './dir1/dir2' => 11}

    manifest.files.each do |stream, basename, size|
      refute(seen[stream].include?(basename), "each_file repeated #{stream}/#{basename}")
      assert_equal(expected_sizes[stream], size, "wrong size for #{stream}/#{basename}")
      seen[stream] << basename
    end

    assert_equal(%w(. ./dir1 ./dir1/dir2), seen.keys)

    seen.each_pair do |stream, basenames|
      assert_equal(%w(file1), basenames.sort, "wrong file list for #{stream}")
    end
  end

  [[false, nil],
   [false, '+0'],
   [false, 'd41d8cd98f00b204e9800998ecf8427'],
   [false, 'd41d8cd98f00b204e9800998ecf8427+0'],
   [false, 'd41d8cd98f00b204e9800998ecf8427e0'],
   [false, 'd41d8cd98f00b204e9800998ecf8427e0+0'],
   [false, 'd41d8cd98f00b204e9800998ecf8427e+0 '],
   [false, "d41d8cd98f00b204e9800998ecf8427e+0\n"],
   [false, ' d41d8cd98f00b204e9800998ecf8427e+0'],
   [false, 'd41d8cd98f00b204e9800998ecf8427e+K+0'],
   [false, 'd41d8cd98f00b204e9800998ecf8427e+0+0'],
   [false, 'd41d8cd98f00b204e9800998ecf8427e++'],
   [false, 'd41d8cd98f00b204e9800998ecf8427e+0+K+'],
   [false, 'd41d8cd98f00b204e9800998ecf8427e+0++K'],
   [false, 'd41d8cd98f00b204e9800998ecf8427e+0+K++'],
   [false, 'd41d8cd98f00b204e9800998ecf8427e+0+K++Z'],
   [true, 'd41d8cd98f00b204e9800998ecf8427e', nil,nil,nil],
   [true, 'd41d8cd98f00b204e9800998ecf8427e+0', '+0','0',nil],
   [true, 'd41d8cd98f00b204e9800998ecf8427e+0+Fizz+Buzz','+0','0','+Fizz+Buzz'],
   [true, 'd41d8cd98f00b204e9800998ecf8427e+Fizz+Buzz', nil,nil,'+Fizz+Buzz'],
   [true, 'd41d8cd98f00b204e9800998ecf8427e+0+Ad41d8cd98f00b204e9800998ecf8427e00000000+Foo', '+0','0','+Ad41d8cd98f00b204e9800998ecf8427e00000000+Foo'],
   [true, 'd41d8cd98f00b204e9800998ecf8427e+Ad41d8cd98f00b204e9800998ecf8427e00000000+Foo', nil,nil,'+Ad41d8cd98f00b204e9800998ecf8427e00000000+Foo'],
   [true, 'd41d8cd98f00b204e9800998ecf8427e+0+Z', '+0','0','+Z'],
   [true, 'd41d8cd98f00b204e9800998ecf8427e+Z', nil,nil,'+Z'],
  ].each do |ok, locator, match2, match3, match4|
    define_method "test_LOCATOR_REGEXP_on_#{locator.inspect}" do
      match = Keep::Locator::LOCATOR_REGEXP.match locator
      assert_equal ok, !!match
      if ok
        assert_equal match2, match[2]
        assert_equal match3, match[3]
        assert_equal match4, match[4]
      end
    end
    define_method "test_parse_method_on_#{locator.inspect}" do
      loc = Keep::Locator.parse locator
      if !ok
        assert_nil loc
      else
        refute_nil loc
        assert loc.is_a?(Keep::Locator)
        #assert loc.hash
        #assert loc.size
        #assert loc.hints.is_a?(Array)
      end
    end
  end

  [
    [false, nil, "No manifest found"],
    [true, ""],
    [false, " ", "Invalid manifest: does not end with newline"],
    [true, ". d41d8cd98f00b204e9800998ecf8427e 0:0:abc.txt\n"],
    [true, ". d41d8cd98f00b204e9800998ecf8427e+0 0:0:abc.txt\n"],
    [true, ". d41d8cd98f00b204e9800998ecf8427e a41d8cd98f00b204e9800998ecf8427e+0 0:0:abc.txt\n"], # 2 locators
    [true, ". d41d8cd98f00b204e9800998ecf8427e+0 0:0:foo/bar.txt\n"],
    [true, ". d41d8cd98f00b204e9800998ecf8427e+0 0:0:.foo.txt\n"],
    [true, ". d41d8cd98f00b204e9800998ecf8427e+0 0:0:.foo\n"],
    [true, ". d41d8cd98f00b204e9800998ecf8427e+0 0:0:...\n"],
    [true, ". d41d8cd98f00b204e9800998ecf8427e+0 0:0:.../.foo./.../bar\n"],
    [true, ". d41d8cd98f00b204e9800998ecf8427e+0 0:0:foo/...\n"],
    [true, ". d41d8cd98f00b204e9800998ecf8427e+0 0:0:foo/.../bar\n"],
    [true, ". d41d8cd98f00b204e9800998ecf8427e+0 0:0:foo/.bar/baz.txt\n"],
    [true, ". d41d8cd98f00b204e9800998ecf8427e+0 0:0:foo/bar./baz.txt\n"],
    [true, ". d41d8cd98f00b204e9800998ecf8427e+0 000000000000000000000000000000:0777:foo.txt\n"],
    [true, ". d41d8cd98f00b204e9800998ecf8427e+0 0:0:0:0\n"],
    [true, ". d41d8cd98f00b204e9800998ecf8427e+0 0:0:\\040\n"],
    [true, ". 00000000000000000000000000000000+0 0:0:0\n"],
    [true, ". 00000000000000000000000000000000+0 0:0:d41d8cd98f00b204e9800998ecf8427e+0+Ad41d8cd98f00b204e9800998ecf8427e00000000@ffffffff\n"],
    [true, ". d41d8cd98f00b204e9800998ecf8427e+0+Ad41d8cd98f00b204e9800998ecf8427e00000000@ffffffff 0:0:empty.txt\n"],
    [true, "./empty_dir d41d8cd98f00b204e9800998ecf8427e+0 0:0:.\n"],
    [false, '. d41d8cd98f00b204e9800998ecf8427e 0:0:abc.txt',
      "Invalid manifest: does not end with newline"],
    [false, "abc d41d8cd98f00b204e9800998ecf8427e 0:0:abc.txt\n",
      "invalid stream name \"abc\""],
    [false, "abc/./foo d41d8cd98f00b204e9800998ecf8427e 0:0:abc.txt\n",
      "invalid stream name \"abc/./foo\""],
    [false, "./abc/../foo d41d8cd98f00b204e9800998ecf8427e 0:0:abc.txt\n",
      "invalid stream name \"./abc/../foo\""],
    [false, "./abc/. d41d8cd98f00b204e9800998ecf8427e 0:0:abc.txt\n",
      "invalid stream name \"./abc/.\""],
    [false, "./abc/.. d41d8cd98f00b204e9800998ecf8427e 0:0:abc.txt\n",
      "invalid stream name \"./abc/..\""],
    [false, "./abc/./foo d41d8cd98f00b204e9800998ecf8427e 0:0:abc.txt\n",
      "invalid stream name \"./abc/./foo\""],
    # non-empty '.'-named file tokens aren't acceptable. Empty ones are used as empty dir placeholders.
    [false, ". 8cf8463b34caa8ac871a52d5dd7ad1ef+1 0:1:.\n",
      "invalid file token \"0:1:.\""],
    [false, ". d41d8cd98f00b204e9800998ecf8427e 0:0:..\n",
      "invalid file token \"0:0:..\""],
    [false, ". d41d8cd98f00b204e9800998ecf8427e 0:0:./abc.txt\n",
      "invalid file token \"0:0:./abc.txt\""],
    [false, ". d41d8cd98f00b204e9800998ecf8427e 0:0:../abc.txt\n",
      "invalid file token \"0:0:../abc.txt\""],
    [false, ". d41d8cd98f00b204e9800998ecf8427e 0:0:abc.txt/.\n",
      "invalid file token \"0:0:abc.txt/.\""],
    [false, ". d41d8cd98f00b204e9800998ecf8427e 0:0:abc.txt/..\n",
      "invalid file token \"0:0:abc.txt/..\""],
    [false, ". d41d8cd98f00b204e9800998ecf8427e 0:0:a/./bc.txt\n",
      "invalid file token \"0:0:a/./bc.txt\""],
    [false, ". d41d8cd98f00b204e9800998ecf8427e 0:0:a/../bc.txt\n",
      "invalid file token \"0:0:a/../bc.txt\""],
    [false, "d41d8cd98f00b204e9800998ecf8427e+0 0:0:abc.txt\n",
      "invalid stream name \"d41d8cd98f00b204e9800998ecf8427e+0\""],
    [false, ". d41d8cd98f00b204e9800998ecf8427 0:0:abc.txt\n",
      "invalid locator \"d41d8cd98f00b204e9800998ecf8427\""],
    [false, ". d41d8cd98f00b204e9800998ecf8427e\n",
      "Manifest invalid for stream 1: no file tokens"],
    [false, ". d41d8cd98f00b204e9800998ecf8427e+0 0:0:abc.txt\n/dir1 d41d8cd98f00b204e9800998ecf842 0:0:abc.txt\n",
      "Manifest invalid for stream 2: missing or invalid stream name \"/dir1\""],
    [false, ". d41d8cd98f00b204e9800998ecf8427e+0 0:0:abc.txt\n./dir1 d41d8cd98f00b204e9800998ecf842 0:0:abc.txt\n",
      "Manifest invalid for stream 2: missing or invalid locator \"d41d8cd98f00b204e9800998ecf842\""],
    [false, ". d41d8cd98f00b204e9800998ecf8427e+0 0:0:abc.txt\n./dir1 a41d8cd98f00b204e9800998ecf8427e+0 abc.txt\n",
      "Manifest invalid for stream 2: invalid file token \"abc.txt\""],
    [false, ". d41d8cd98f00b204e9800998ecf8427e+0 0:0:abc.txt\n./dir1 a41d8cd98f00b204e9800998ecf8427e+0 0:abc.txt\n",
      "Manifest invalid for stream 2: invalid file token \"0:abc.txt\""],
    [false, ". d41d8cd98f00b204e9800998ecf8427e+0 0:0:abc.txt\n./dir1 a41d8cd98f00b204e9800998ecf8427e+0 0:0:abc.txt xyz.txt\n",
      "Manifest invalid for stream 2: invalid file token \"xyz.txt\""],
    [false, ". d41d8cd98f00b204e9800998ecf8427e+0 0:0:foo.txt d41d8cd98f00b204e9800998ecf8427e+0\n",
      "Manifest invalid for stream 1: invalid file token \"d41d8cd98f00b204e9800998ecf8427e+0\""],
    [false, ". d41d8cd98f00b204e9800998ecf8427e+0 0:0:\n",
      "Manifest invalid for stream 1: invalid file token \"0:0:\""],
    [false, ". d41d8cd98f00b204e9800998ecf8427e+0\n",
      "Manifest invalid for stream 1: no file tokens"],
    [false, ". 0:0:foo.txt d41d8cd98f00b204e9800998ecf8427e+0\n",
      "Manifest invalid for stream 1: missing or invalid locator \"0:0:foo.txt\""],
    [false, ". 0:0:foo.txt\n",
      "Manifest invalid for stream 1: missing or invalid locator \"0:0:foo.txt\""],
    [false, ".\n", "Manifest invalid for stream 1: missing or invalid locator"],
    [false, ".", "Invalid manifest: does not end with newline"],
    [false, ". \n", "Manifest invalid for stream 1: missing or invalid locator"],
    [false, ".  \n", "Manifest invalid for stream 1: missing or invalid locator"],
    [false, " . d41d8cd98f00b204e9800998ecf8427e+0 0:0:foo.txt\n",
      "Manifest invalid for stream 1: missing or invalid stream name"],
    [false, ". d41d8cd98f00b204e9800998ecf8427e+0 0:0:foo.txt \n",
      "stream 1: trailing space"],
   # TAB and other tricky whitespace characters:
    [false, "\v. d41d8cd98f00b204e9800998ecf8427e+0 0:0:foo.txt\n",
      "Manifest invalid for stream 1: missing or invalid stream name \"\\v."],
    [false, "./foo\vbar d41d8cd98f00b204e9800998ecf8427e+0 0:0:foo.txt\n",
      "Manifest invalid for stream 1: missing or invalid stream name \"./foo\\vbar"],
    [false, "\t. d41d8cd98f00b204e9800998ecf8427e+0 0:0:foo.txt\n",
      "Manifest invalid for stream 1: missing or invalid stream name \"\\t"],
    [false, ".\td41d8cd98f00b204e9800998ecf8427e+0 0:0:foo.txt\n",
      "Manifest invalid for stream 1: missing or invalid stream name \".\\t"],
    [false, ". d41d8cd98f00b204e9800998ecf8427e+0 0:0:foo.txt\t\n",
      "stream 1: invalid file token \"0:0:foo.txt\\t\""],
    [false, ". d41d8cd98f00b204e9800998ecf8427e+0\t 0:0:foo.txt\n",
      "stream 1: missing or invalid locator \"d41d8cd98f00b204e9800998ecf8427e+0\\t\""],
    [false, "./foo\tbar d41d8cd98f00b204e9800998ecf8427e+0 0:0:foo.txt\n",
      "stream 1: missing or invalid stream name \"./foo\\tbar\""],
    # other whitespace errors:
    [false, ". d41d8cd98f00b204e9800998ecf8427e+0  0:0:foo.txt\n",
      "Manifest invalid for stream 1: invalid file token \"\""],
    [false, ". d41d8cd98f00b204e9800998ecf8427e+0 0:0:foo.txt\n \n",
      "Manifest invalid for stream 2: missing stream name"],
    [false, ". d41d8cd98f00b204e9800998ecf8427e+0 0:0:foo.txt\n\n",
      "Manifest invalid for stream 2: missing stream name"],
    [false, ". d41d8cd98f00b204e9800998ecf8427e+0 0:0:foo.txt\n ",
      "Invalid manifest: does not end with newline"],
    [false, "\n. d41d8cd98f00b204e9800998ecf8427e+0 0:0:foo.txt\n",
      "Manifest invalid for stream 1: missing stream name"],
    [false, " \n. d41d8cd98f00b204e9800998ecf8427e+0 0:0:foo.txt\n",
      "Manifest invalid for stream 1: missing stream name"],
    # empty file and stream name components:
    [false, ". d41d8cd98f00b204e9800998ecf8427e+0 0:0:/foo.txt\n",
      "Manifest invalid for stream 1: invalid file token \"0:0:/foo.txt\""],
    [false, "./ d41d8cd98f00b204e9800998ecf8427e+0 0:0:foo.txt\n",
      "Manifest invalid for stream 1: missing or invalid stream name \"./\""],
    [false, ".//foo d41d8cd98f00b204e9800998ecf8427e+0 0:0:foo.txt\n",
      "Manifest invalid for stream 1: missing or invalid stream name \".//foo\""],
    [false, "./foo/ d41d8cd98f00b204e9800998ecf8427e+0 0:0:foo.txt\n",
      "Manifest invalid for stream 1: missing or invalid stream name \"./foo/\""],
    [false, "./foo//bar d41d8cd98f00b204e9800998ecf8427e+0 0:0:foo.txt\n",
      "Manifest invalid for stream 1: missing or invalid stream name \"./foo//bar\""],
    [false, ". d41d8cd98f00b204e9800998ecf8427e+0 0:0:foo//bar.txt\n",
      "Manifest invalid for stream 1: invalid file token \"0:0:foo//bar.txt\""],
    [false, ". d41d8cd98f00b204e9800998ecf8427e+0 0:0:foo/\n",
      "Manifest invalid for stream 1: invalid file token \"0:0:foo/\""],
    # escaped chars
    [true, "./empty_dir d41d8cd98f00b204e9800998ecf8427e+0 0:0:\\056\n"],
    [false, "./empty_dir d41d8cd98f00b204e9800998ecf8427e+0 0:0:\\056\\056\n",
      "Manifest invalid for stream 1: invalid file token \"0:0:\\\\056\\\\056\""],
    [false, "./empty_dir d41d8cd98f00b204e9800998ecf8427e+0 0:0:\\056\\056\\057foo\n",
      "Manifest invalid for stream 1: invalid file token \"0:0:\\\\056\\\\056\\\\057foo\""],
    [false, "./empty_dir d41d8cd98f00b204e9800998ecf8427e+0 0\\0720\\072foo\n",
      "Manifest invalid for stream 1: invalid file token \"0\\\\0720\\\\072foo\""],
    [false, "./empty_dir d41d8cd98f00b204e9800998ecf8427e+0 \\060:\\060:foo\n",
      "Manifest invalid for stream 1: invalid file token \"\\\\060:\\\\060:foo\""],
    [true, ". d41d8cd98f00b204e9800998ecf8427e+0 0:0:foo\\057bar\n"],
    [true, ". d41d8cd98f00b204e9800998ecf8427e+0 0:0:\\072\n"],
    [true, ".\\057Data d41d8cd98f00b204e9800998ecf8427e+0 0:0:foo\n"],
    [true, "\\056\\057Data d41d8cd98f00b204e9800998ecf8427e+0 0:0:foo\n"],
    [true, "./\\134444 d41d8cd98f00b204e9800998ecf8427e+0 0:0:foo\n"],
    [false, "./\\\\444 d41d8cd98f00b204e9800998ecf8427e+0 0:0:foo\n",
      "Manifest invalid for stream 1: missing or invalid stream name \"./\\\\\\\\444\""],
    [true, "./\\011foo d41d8cd98f00b204e9800998ecf8427e+0 0:0:foo\n"],
    [false, "./\\011/.. d41d8cd98f00b204e9800998ecf8427e+0 0:0:foo\n",
      "Manifest invalid for stream 1: missing or invalid stream name \"./\\\\011/..\""],
    [false, ".\\056\\057 d41d8cd98f00b204e9800998ecf8427e+0 0:0:foo\n",
      "Manifest invalid for stream 1: missing or invalid stream name \".\\\\056\\\\057\""],
    [false, ".\\057\\056 d41d8cd98f00b204e9800998ecf8427e+0 0:0:foo\n",
      "Manifest invalid for stream 1: missing or invalid stream name \".\\\\057\\\\056\""],
    [false, ".\\057Data d41d8cd98f00b204e9800998ecf8427e+0 0:0:foo\\444\n",
      "Manifest invalid for stream 1: >8-bit encoded chars not allowed on file token \"0:0:foo\\\\444\""],
    [false, "./\\444 d41d8cd98f00b204e9800998ecf8427e+0 0:0:foo\n",
      "Manifest invalid for stream 1: >8-bit encoded chars not allowed on stream token \"./\\\\444\""],
    [false, "./\tfoo d41d8cd98f00b204e9800998ecf8427e+0 0:0:foo\n",
      "Manifest invalid for stream 1: missing or invalid stream name \"./\\tfoo\""],
    [false, "./foo\\ d41d8cd98f00b204e9800998ecf8427e+0 0:0:foo\n",
      "Manifest invalid for stream 1: missing or invalid stream name \"./foo\\\\\""],
    [false, "./foo\\r d41d8cd98f00b204e9800998ecf8427e+0 0:0:foo\n",
      "Manifest invalid for stream 1: missing or invalid stream name \"./foo\\\\r\""],
    [false, "./foo\\444 d41d8cd98f00b204e9800998ecf8427e+0 0:0:foo\n",
      "Manifest invalid for stream 1: >8-bit encoded chars not allowed on stream token \"./foo\\\\444\""],
    [false, "./foo\\888 d41d8cd98f00b204e9800998ecf8427e+0 0:0:foo\n",
      "Manifest invalid for stream 1: missing or invalid stream name \"./foo\\\\888\""],
    [false, ". d41d8cd98f00b204e9800998ecf8427e+0 0:0:foo\\\n",
      "Manifest invalid for stream 1: invalid file token \"0:0:foo\\\\\""],
    [false, ". d41d8cd98f00b204e9800998ecf8427e+0 0:0:foo\\r\n",
      "Manifest invalid for stream 1: invalid file token \"0:0:foo\\\\r\""],
    [false, ". d41d8cd98f00b204e9800998ecf8427e+0 0:0:foo\\444\n",
      "Manifest invalid for stream 1: >8-bit encoded chars not allowed on file token \"0:0:foo\\\\444\""],
    [false, ". d41d8cd98f00b204e9800998ecf8427e+0 0:0:foo\\888\n",
      "Manifest invalid for stream 1: invalid file token \"0:0:foo\\\\888\""],
    [false, ". d41d8cd98f00b204e9800998ecf8427e+0 0:0:foo\\057/bar\n",
      "Manifest invalid for stream 1: invalid file token \"0:0:foo\\\\057/bar\""],
    [false, ".\\057/Data d41d8cd98f00b204e9800998ecf8427e+0 0:0:foo\n",
      "Manifest invalid for stream 1: missing or invalid stream name \".\\\\057/Data\""],
    [true, "./Data\\040Folder d41d8cd98f00b204e9800998ecf8427e+0 0:0:foo\n"],
    [false, ". d41d8cd98f00b204e9800998ecf8427e+0 0:0:\\057foo/bar\n",
      "Manifest invalid for stream 1: invalid file token \"0:0:\\\\057foo/bar\""],
    [true, ". d41d8cd98f00b204e9800998ecf8427e+0 0:0:\\134057foo/bar\n"],
    [false, ". d41d8cd98f00b204e9800998ecf8427e+0 \\040:\\040:foo.txt\n",
      "Manifest invalid for stream 1: invalid file token \"\\\\040:\\\\040:foo.txt\""],
  ].each do |ok, manifest, expected_error=nil|
    define_method "test_validate manifest #{manifest.inspect}" do
      assert_equal ok, Keep::Manifest.valid?(manifest)
      if ok
        assert Keep::Manifest.validate! manifest
      else
        begin
          Keep::Manifest.validate! manifest
        rescue ArgumentError => e
          msg = e.message
        end
        refute_nil msg, "Expected ArgumentError"
        assert msg.include?(expected_error), "Did not find expected error message. Expected: #{expected_error}; Actual: #{msg}"
      end
    end
  end
end
