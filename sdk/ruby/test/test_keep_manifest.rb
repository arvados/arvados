require "minitest/autorun"
require "arvados/keep"

def random_block(size=nil)
  sprintf("%032x+%d", rand(16 ** 32), size || rand(64 * 1024 * 1024))
end

class ManifestTest < Minitest::Test
  SIMPLEST_MANIFEST = ". #{random_block(9)} 0:9:simple.txt\n"
  MULTILEVEL_MANIFEST =
    [". #{random_block(9)} 0:3:file1 3:3:file2 6:3:file3\n",
     "./dir1 #{random_block(9)} 0:3:file1 3:3:file2 6:3:file3\n",
     "./dir1/subdir #{random_block(9)} 0:3:file1 3:3:file2 6:3:file3\n",
     "./dir2 #{random_block(9)} 0:3:file1 3:3:file2 6:3:file3\n"].join("")

  def test_simple_each_stream_array
    manifest = Keep::Manifest.new(SIMPLEST_MANIFEST)
    stream_name, block_s, file = SIMPLEST_MANIFEST.strip.split
    stream_a = manifest.each_stream.to_a
    assert_equal(1, stream_a.size, "wrong number of streams")
    assert_equal(stream_name, stream_a[0][0])
    assert_equal([block_s], stream_a[0][1].map(&:to_s))
    assert_equal([file], stream_a[0][2])
  end

  def test_simple_each_stream_block
    manifest = Keep::Manifest.new(SIMPLEST_MANIFEST)
    result = []
    manifest.each_stream do |stream, blocks, files|
      result << files
    end
    assert_equal([[SIMPLEST_MANIFEST.split.last]], result,
                 "wrong result from each_stream block")
  end

  def test_multilevel_each_stream
    manifest = Keep::Manifest.new(MULTILEVEL_MANIFEST)
    seen = []
    manifest.each_stream do |stream, blocks, files|
      refute(seen.include?(stream),
             "each_stream already yielded stream #{stream}")
      seen << stream
      assert_equal(3, files.size, "wrong file count for stream #{stream}")
    end
    assert_equal(4, seen.size, "wrong number of streams")
  end

  def test_empty_each_stream
    assert_empty(Keep::Manifest.new("").each_stream.to_a)
  end

  def test_backslash_escape_parsing
    m_text = "./dir\\040name #{random_block} 0:0:file\\\\name\\011\\here.txt\n"
    manifest = Keep::Manifest.new(m_text)
    streams = manifest.each_stream.to_a
    assert_equal(1, streams.size, "wrong number of streams with whitespace")
    assert_equal("./dir name", streams.first.first,
                 "wrong stream name with whitespace")
    assert_equal(["0:0:file\\name\t\\here.txt"], streams.first.last,
                 "wrong filename(s) with whitespace")
  end

  def test_simple_each_file_array
    manifest = Keep::Manifest.new(SIMPLEST_MANIFEST)
    assert_equal([[".", "simple.txt", 9]], manifest.each_file.to_a)
  end

  def test_multilevel_each_file
    manifest = Keep::Manifest.new(MULTILEVEL_MANIFEST)
    seen = Hash.new { |this, key| this[key] = [] }
    manifest.each_file do |stream, basename, size|
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

  def test_each_file_handles_filenames_with_colons
    manifest = Keep::Manifest.new(". #{random_block(9)} 0:9:file:test.txt\n")
    assert_equal([[".", "file:test.txt", 9]], manifest.each_file.to_a)
  end
end
