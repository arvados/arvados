require "yaml"

module SDKFixtures
  module StaticMethods
    # SDKFixtures will use these as class methods, and install them as
    # instance methods on the test classes.
    def random_block(size=nil)
      sprintf("%032x+%d", rand(16 ** 32), size || rand(64 * 1024 * 1024))
    end

    def random_blocks(count, size=nil)
      (0...count).map { |_| random_block(size) }
    end
  end

  extend StaticMethods

  def self.included(base)
    base.include(StaticMethods)
  end

  @@fixtures = {}
  def fixtures name
    @@fixtures[name] ||=
      begin
        path = File.
          expand_path("../../../../services/api/test/fixtures/#{name}.yml",
                      __FILE__)
        file = IO.read(path)
        trim_index = file.index('# Test Helper trims the rest of the file')
        file = file[0, trim_index] if trim_index
        YAML.load(file)
      end
  end

  ### Valid manifests
  SIMPLEST_MANIFEST = ". #{random_block(9)} 0:9:simple.txt\n"
  MULTIBLOCK_FILE_MANIFEST =
    [". #{random_block(8)} 0:4:repfile 4:4:uniqfile",
     "./s1 #{random_block(6)} 0:3:repfile 3:3:uniqfile",
     ". #{random_block(8)} 0:7:uniqfile2 7:1:repfile\n"].join("\n")
  MULTILEVEL_MANIFEST =
    [". #{random_block(9)} 0:3:file1 3:3:file2 6:3:file3\n",
     "./dir0 #{random_block(9)} 0:3:file1 3:3:file2 6:3:file3\n",
     "./dir0/subdir #{random_block(9)} 0:3:file1 3:3:file2 6:3:file3\n",
     "./dir1 #{random_block(9)} 0:3:file1 3:3:file2 6:3:file3\n",
     "./dir1/subdir #{random_block(9)} 0:3:file1 3:3:file2 6:3:file3\n",
     "./dir2 #{random_block(9)} 0:3:file1 3:3:file2 6:3:file3\n"].join("")
  COLON_FILENAME_MANIFEST = ". #{random_block(9)} 0:9:file:test.txt\n"
  # Filename is `a a.txt`.
  ESCAPED_FILENAME_MANIFEST = ". #{random_block(9)} 0:9:a\\040\\141.txt\n"
  MANY_ESCAPES_MANIFEST =
    "./dir\\040name #{random_block(9)} 0:9:file\\\\name\\011\\here.txt\n"
  NONNORMALIZED_MANIFEST =
    ["./dir2 #{random_block} 0:0:z 0:0:y 0:0:x",
     "./dir1 #{random_block} 0:0:p 0:0:o 0:0:n\n"].join("\n")

  ### Non-tree manifests
  # These manifests follow the spec, but they express a structure that can't
  # can't be represented by a POSIX filesystem tree.  For example, there's a
  # name conflict between a stream and a filename.
  NAME_CONFLICT_MANIFEST =
    [". #{random_block(9)} 0:9:conflict",
     "./conflict #{random_block} 0:0:name\n"].join("\n")
end
