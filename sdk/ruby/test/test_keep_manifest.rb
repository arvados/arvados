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

  def test_multilevel_collection_with_dirs_in_filenames
    manifest = Keep::Manifest.new(MULTILEVEL_MANIFEST_WITH_DIRS_IN_FILENAMES)

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
end
