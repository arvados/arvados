require "arvados/keep"

module Arv
  class Collection
    def initialize(manifest_text="")
      @tree = CollectionStream.new(".")
      @manifest_text = ""
      import_manifest!(manifest_text)
    end

    def manifest_text
      @manifest_text ||= @tree.manifest_text
    end

    def import_manifest!(manifest_text)
      manifest = Keep::Manifest.new(manifest_text)
      manifest.each_line do |stream_root, locators, file_specs|
        if stream_root.empty? or locators.empty? or file_specs.empty?
          raise ArgumentError.new("manifest text includes malformed line")
        end
        file_specs.map { |s| manifest.split_file_token(s) }.
            each do |file_start, file_len, file_path|
          @tree.file_at(normalize_path(stream_root, file_path)).
            add_range(locators, file_start, file_len)
        end
      end
      if @manifest_text == ""
        @manifest_text = manifest_text
        self
      else
        modified!
      end
    end

    def normalize!
      # We generate normalized manifests, so all we have to do is force
      # regeneration.
      modified!
    end

    def copy!(source, target, source_collection=nil)
      copy(:merge, source, target, source_collection)
    end

    def rename!(source, target)
      copy(:add_copy, source, target) { remove!(source, recursive: true) }
    end

    def remove!(path, opts={})
      stream, name = find(path)
      if name.nil?
        return self if @tree.leaf?
        @tree = CollectionStream.new(".")
      else
        stream.delete(name, opts)
      end
      modified!
    end

    protected

    def find(*parts)
      normpath = normalize_path(*parts)
      if normpath.empty?
        [@tree, nil]
      else
        @tree.find(normpath)
      end
    end

    private

    def copy(copy_method, source, target, source_collection=nil)
      # Find the item at path `source` in `source_collection`, find the
      # destination stream at path `target`, and use `copy_method` to copy
      # the found object there.  If a block is passed in, it will be called
      # right before we do the actual copy, after we confirm that everything
      # is found and can be copied.
      source_collection = self if source_collection.nil?
      src_stream, src_tail = source_collection.find(source)
      dst_stream, dst_tail = find(target)
      if (source_collection.equal?(self) and
          (src_stream.path == dst_stream.path) and (src_tail == dst_tail))
        return self
      elsif src_tail.nil?
        src_item = src_stream
        src_tail = src_stream.name
      else
        src_item = src_stream[src_tail]
      end
      dst_tail ||= src_tail
      check_method = "check_can_#{copy_method}".to_sym
      begin
        # Find out if `target` refers to a stream we should copy into.
        tail_stream = dst_stream[dst_tail]
        tail_stream.send(check_method, src_item, src_tail)
      rescue Errno::ENOENT, Errno::ENOTDIR
        # It does not.  Check that we can copy `source` to the full
        # path specified by `target`.
        dst_stream.send(check_method, src_item, dst_tail)
        target_name = dst_tail
      else
        # Yes, `target` is a stream.  Copy the item at `source` into it with
        # the same name.
        dst_stream = tail_stream
        target_name = src_tail
      end
      # At this point, we know the operation will work.  Call any block as
      # a pre-copy hook.
      if block_given?
        yield
        # Re-find the destination stream, in case the block removed
        # the original (that's how rename is implemented).
        dst_path = normalize_path(dst_stream.path)
        if dst_path.empty?
          dst_stream = @tree
        else
          dst_stream = @tree.stream_at(dst_path)
        end
      end
      dst_stream.send(copy_method, src_item, target_name)
      modified!
    end

    def modified!
      @manifest_text = nil
      self
    end

    def normalize_path(*parts)
      path = File.join(*parts)
      raise ArgumentError.new("empty path") if path.empty?
      path.sub(/^\.(\/|$)/, "")
    end

    class CollectionItem
      attr_reader :path, :name

      def initialize(path)
        @path = path
        @name = File.basename(path)
      end
    end

    LocatorRange = Struct.new(:locators, :start_pos, :length)

    class CollectionFile < CollectionItem
      def initialize(path)
        super
        @ranges = []
      end

      def self.human_name
        "file"
      end

      def leaf?
        true
      end

      def add_range(locators, start_pos, length)
        # Given an array of locators, and this file's start position and
        # length within them, store a LocatorRange with information about
        # the locators actually used.
        loc_sizes = locators.map { |s| Keep::Locator.parse(s).size.to_i }
        start_index, start_pos = loc_size_index(loc_sizes, start_pos, 0, :>=)
        end_index, _ = loc_size_index(loc_sizes, length, start_index, :>)
        @ranges << LocatorRange.
          new(locators[start_index..end_index], start_pos, length)
      end

      def each_range(&block)
        @ranges.each(&block)
      end

      def check_can_add_copy(src_item, name)
        raise Errno::ENOTDIR.new(path)
      end

      alias_method :check_can_merge, :check_can_add_copy

      def copy_named(copy_path)
        copy = self.class.new(copy_path)
        each_range { |range| copy.add_range(*range) }
        copy
      end

      private

      def loc_size_index(loc_sizes, length, index, comp_op)
        # Pass in an array of locator size hints (integers).  Starting from
        # `index`, step through the size array until they provide a number
        # of bytes that is `comp_op` (:>= or :>) to `length`.  Return the
        # index of the end locator and the amount of data to read from it.
        while length.send(comp_op, loc_sizes[index])
          index += 1
          length -= loc_sizes[index]
        end
        [index, length]
      end
    end

    class CollectionStream < CollectionItem
      def initialize(path)
        super
        @items = {}
      end

      def self.human_name
        "stream"
      end

      def leaf?
        items.empty?
      end

      def [](key)
        items[key] or
          raise Errno::ENOENT.new("%p not found in %p" % [key, path])
      end

      def delete(name, opts={})
        item = self[name]
        if item.leaf? or opts[:recursive]
          items.delete(name)
        else
          raise Errno::ENOTEMPTY.new(path)
        end
      end

      def find(find_path)
        # Given a POSIX-style path, return the CollectionStream that
        # contains the object at that path, and the name of the object
        # inside it.
        components = find_path.split("/")
        tail = components.pop
        [components.reduce(self, :[]), tail]
      end

      def stream_at(find_path)
        key, rest = find_path.split("/", 2)
        next_stream = get_or_new(key, CollectionStream)
        if rest.nil?
          next_stream
        else
          next_stream.stream_at(rest)
        end
      end

      def file_at(find_path)
        stream_path, _, file_name = find_path.rpartition("/")
        if stream_path.empty?
          get_or_new(file_name, CollectionFile)
        else
          stream_at(stream_path).file_at(file_name)
        end
      end

      def manifest_text
        # Return a string with the normalized manifest text for this stream,
        # including all substreams.
        file_keys, stream_keys = items.keys.sort.partition do |key|
          items[key].is_a?(CollectionFile)
        end
        my_line = StreamManifest.new(path)
        file_keys.each do |file_name|
          my_line.add_file(items[file_name])
        end
        sub_lines = stream_keys.map do |sub_name|
          items[sub_name].manifest_text
        end
        my_line.to_s + sub_lines.join("")
      end

      def check_can_add_copy(src_item, key)
        if existing = check_can_merge(src_item, key) and not existing.leaf?
          raise Errno::ENOTEMPTY.new(existing.path)
        end
      end

      def check_can_merge(src_item, key)
        if existing = items[key] and (existing.class != src_item.class)
          raise Errno::ENOTDIR.new(existing.path)
        end
        existing
      end

      def add_copy(src_item, key)
        items[key] = src_item.copy_named("#{path}/#{key}")
      end

      def merge(src_item, key)
        # Do a recursive copy of the collection item `src_item` to destination
        # `key`.  If a simple copy is safe, do that; otherwise, recursively
        # merge the contents of the stream `src_item` into the stream at
        # `key`.
        begin
          check_can_add_copy(src_item, key)
          add_copy(src_item, key)
        rescue Errno::ENOTEMPTY
          dest = self[key]
          error = nil
          # Copy as much as possible, then raise any error encountered.
          src_item.items.each_pair do |sub_key, sub_item|
            begin
              dest.merge(sub_item, sub_key)
            rescue Errno::ENOTDIR => error
            end
          end
          raise error unless error.nil?
        end
      end

      def copy_named(copy_path)
        copy = self.class.new(copy_path)
        items.each_pair do |key, item|
          copy.add_copy(item, key)
        end
        copy
      end

      protected

      attr_reader :items

      private

      def get_or_new(key, klass)
        # Return the collection item at `key` and ensure that it's a `klass`.
        # If `key` does not exist, create a new `klass` there.
        # If the value for `key` is not a `klass`, raise an ArgumentError.
        item = items[key]
        if item.nil?
          items[key] = klass.new("#{path}/#{key}")
        elsif not item.is_a?(klass)
          raise ArgumentError.
            new("in stream %p, %p is a %s, not a %s" %
                [path, key, items[key].class.human_name, klass.human_name])
        else
          item
        end
      end
    end

    class StreamManifest
      # Build a manifest text for a single stream, without substreams.

      def initialize(name)
        @name = name
        @locators = []
        @loc_sizes = []
        @file_specs = []
      end

      def add_file(coll_file)
        coll_file.each_range do |range|
          add(coll_file.name, *range)
        end
      end

      def to_s
        if @file_specs.empty?
          ""
        else
          "%s %s %s\n" % [escape_name(@name), @locators.join(" "),
                          @file_specs.join(" ")]
        end
      end

      private

      def add(file_name, loc_a, file_start, file_len)
        # Ensure that the locators in loc_a appear in this locator in sequence,
        # adding as few as possible.  Save a new file spec based on those
        # locators' position.
        loc_size = @locators.size
        add_size = loc_a.size
        loc_ii = 0
        add_ii = 0
        while (loc_ii < loc_size) and (add_ii < add_size)
          if @locators[loc_ii] == loc_a[add_ii]
            add_ii += 1
          else
            add_ii = 0
          end
          loc_ii += 1
        end
        loc_ii -= add_ii
        to_add = loc_a[add_ii, add_size] || []
        @locators += to_add
        @loc_sizes += to_add.map { |s| Keep::Locator.parse(s).size.to_i }
        start = @loc_sizes[0, loc_ii].reduce(0, &:+) + file_start
        @file_specs << "#{start}:#{file_len}:#{escape_name(file_name)}"
      end

      def escape_name(name)
        name.gsub(/\\/, "\\\\\\\\").gsub(/\s/) do |s|
          s.each_byte.map { |c| "\\%03o" % c }.join("")
        end
      end
    end
  end
end
