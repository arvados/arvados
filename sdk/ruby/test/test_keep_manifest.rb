require "minitest/autorun"
require "arvados/keep"

def random_block(size=nil)
  sprintf("%032x+%d", rand(16 ** 32), size || rand(64 * 1024 * 1024))
end

class ManifestTest < Minitest::Test
  SIMPLEST_MANIFEST = ". #{random_block(9)} 0:9:simple.txt\n"
  MULTIBLOCK_FILE_MANIFEST =
    [". #{random_block(8)} 0:4:repfile 4:4:uniqfile",
     "./s1 #{random_block(6)} 0:3:repfile 3:3:uniqfile",
     ". #{random_block(8)} 0:7:uniqfile2 7:1:repfile\n"].join("\n")
  MULTILEVEL_MANIFEST =
    [". #{random_block(9)} 0:3:file1 3:3:file2 6:3:file3\n",
     "./dir1 #{random_block(9)} 0:3:file1 3:3:file2 6:3:file3\n",
     "./dir1/subdir #{random_block(9)} 0:3:file1 3:3:file2 6:3:file3\n",
     "./dir2 #{random_block(9)} 0:3:file1 3:3:file2 6:3:file3\n"].join("")

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
    assert_equal(4, seen.size, "wrong number of streams")
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
    m_text = "./dir\\040name #{random_block} 0:0:file\\\\name\\011\\here.txt\n"
    manifest = Keep::Manifest.new(m_text)
    streams = manifest.each_line.to_a
    assert_equal(1, streams.size, "wrong number of streams with whitespace")
    assert_equal("./dir name", streams.first.first,
                 "wrong stream name with whitespace")
    assert_equal(["0:0:file\\name\t\\here.txt"], streams.first.last,
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
    manifest = Keep::Manifest.new(". #{random_block(9)} 0:9:file:test.txt\n")
    assert_equal([[".", "file:test.txt", 9]], manifest.files)
  end

  def test_files_with_escape_sequence_in_filename
    manifest = Keep::Manifest.new(". #{random_block(9)} 0:9:a\\040\\141.txt\n")
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
    manifest = Keep::Manifest.new(". #{random_block(3)} 0:3:a\\040b\\040c\n")
    assert(manifest.has_file?("./a b c"), "one-arg 'a b c' not found")
    assert(manifest.has_file?(".", "a b c"), "two-arg 'a b c' not found")
    refute(manifest.has_file?("a\\040b\\040c"), "one-arg unescaped found")
    refute(manifest.has_file?(".", "a\\040b\\040c"), "two-arg unescaped found")
  end
end
