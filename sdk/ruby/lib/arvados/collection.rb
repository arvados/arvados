require "arvados/keep"

module Arv
  class Collection
    def initialize(manifest_text="")
      @manifest_text = manifest_text
      @modified = false
      @root = CollectionRoot.new
      manifest = Keep::Manifest.new(manifest_text)
      manifest.each_line do |stream_root, locators, file_specs|
        if stream_root.empty? or locators.empty? or file_specs.empty?
          raise ArgumentError.new("manifest text includes malformed line")
        end
        loc_list = LocatorList.new(locators)
        file_specs.map { |s| manifest.split_file_token(s) }.
            each do |file_start, file_len, file_path|
          @root.file_at(normalize_path(stream_root, file_path)).
            add_segment(loc_list.segment(file_start, file_len))
        end
      end
    end

    def manifest_text
      @manifest_text ||= @root.manifest_text
    end

    def modified?
      @modified
    end

    def unmodified
      @modified = false
      self
    end

    def normalize
      @manifest_text = @root.manifest_text
      self
    end

    def cp_r(source, target, source_collection=nil)
      opts = {descend_target: !source.end_with?("/")}
      copy(:merge, source.chomp("/"), target, source_collection, opts)
    end

    def exist?(path)
      begin
        substream, item = find(path)
        not (substream.leaf? or substream[item].nil?)
      rescue Errno::ENOENT, Errno::ENOTDIR
        false
      end
    end

    def rename(source, target)
      copy(:add_copy, source, target) { rm_r(source) }
    end

    def rm(source)
      remove(source)
    end

    def rm_r(source)
      remove(source, recursive: true)
    end

    protected

    def find(*parts)
      @root.find(normalize_path(*parts))
    end

    private

    def modified
      @manifest_text = nil
      @modified = true
      self
    end

    def normalize_path(*parts)
      path = File.join(*parts)
      if path.empty?
        raise ArgumentError.new("empty path")
      elsif (path == ".") or path.start_with?("./")
        path
      else
        "./#{path}"
      end
    end

    def copy(copy_method, source, target, source_collection=nil, opts={})
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
      end
      src_item = src_stream[src_tail]
      dst_tail ||= src_tail
      check_method = "check_can_#{copy_method}".to_sym
      target_name = nil
      if opts.fetch(:descend_target, true)
        begin
          # Find out if `target` refers to a stream we should copy into.
          tail_stream = dst_stream[dst_tail]
          tail_stream.send(check_method, src_item, src_tail)
          # Yes it does.  Copy the item at `source` into it with the same name.
          dst_stream = tail_stream
          target_name = src_tail
        rescue Errno::ENOENT, Errno::ENOTDIR
          # It does not.  We'll fall back to writing to `target` below.
        end
      end
      if target_name.nil?
        dst_stream.send(check_method, src_item, dst_tail)
        target_name = dst_tail
      end
      # At this point, we know the operation will work.  Call any block as
      # a pre-copy hook.
      if block_given?
        yield
        # Re-find the destination stream, in case the block removed
        # the original (that's how rename is implemented).
        dst_stream = @root.stream_at(dst_stream.path)
      end
      dst_stream.send(copy_method, src_item, target_name)
      modified
    end

    def remove(path, opts={})
      stream, name = find(path)
      stream.delete(name, opts)
      modified
    end

    LocatorSegment = Struct.new(:locators, :start_pos, :length)

    class LocatorRange < Range
      attr_reader :locator

      def initialize(loc_s, start)
        @locator = loc_s
        range_end = start + Keep::Locator.parse(loc_s).size.to_i
        super(start, range_end, false)
      end
    end

    class LocatorList
      # LocatorList efficiently builds LocatorSegments from a stream manifest.
      def initialize(locators)
        next_start = 0
        @ranges = locators.map do |loc_s|
          new_range = LocatorRange.new(loc_s, next_start)
          next_start = new_range.end
          new_range
        end
      end

      def segment(start_pos, length)
        # Return a LocatorSegment that captures `length` bytes from `start_pos`.
        start_index = search_for_byte(start_pos)
        if length == 0
          end_index = start_index
        else
          end_index = search_for_byte(start_pos + length - 1, start_index)
        end
        seg_ranges = @ranges[start_index..end_index]
        LocatorSegment.new(seg_ranges.map(&:locator),
                           start_pos - seg_ranges.first.begin,
                           length)
      end

      private

      def search_for_byte(target, start_index=0)
        # Do a binary search for byte `target` in the list of locators,
        # starting from `start_index`.  Return the index of the range in
        # @ranges that contains the byte.
        lo = start_index
        hi = @ranges.size
        loop do
          ii = (lo + hi) / 2
          range = @ranges[ii]
          if range.include?(target)
            return ii
          elsif ii == lo
            raise RangeError.new("%i not in segment" % target)
          elsif target < range.begin
            hi = ii
          else
            lo = ii
          end
        end
      end
    end

    class CollectionItem
      attr_reader :path, :name

      def initialize(path)
        @path = path
        @name = File.basename(path)
      end
    end

    class CollectionFile < CollectionItem
      def initialize(path)
        super
        @segments = []
      end

      def self.human_name
        "file"
      end

      def file?
        true
      end

      def leaf?
        true
      end

      def add_segment(segment)
        @segments << segment
      end

      def each_segment(&block)
        @segments.each(&block)
      end

      def check_can_add_copy(src_item, name)
        raise Errno::ENOTDIR.new(path)
      end

      alias_method :check_can_merge, :check_can_add_copy

      def copy_named(copy_path)
        copy = self.class.new(copy_path)
        each_segment { |segment| copy.add_segment(segment) }
        copy
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

      def file?
        false
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
        if item.file? or opts[:recursive]
          items.delete(name)
        else
          raise Errno::EISDIR.new(path)
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
          items[key].file?
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
        self[key] = src_item.copy_named("#{path}/#{key}")
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
          # Start with streams for a depth-first merge.
          src_items = src_item.items.each_pair.sort_by do |_, sub_item|
            (sub_item.file?) ? 1 : 0
          end
          src_items.each do |sub_key, sub_item|
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

      def []=(key, item)
        items[key] = item
      end

      def get_or_new(key, klass)
        # Return the collection item at `key` and ensure that it's a `klass`.
        # If `key` does not exist, create a new `klass` there.
        # If the value for `key` is not a `klass`, raise an ArgumentError.
        item = items[key]
        if item.nil?
          self[key] = klass.new("#{path}/#{key}")
        elsif not item.is_a?(klass)
          raise ArgumentError.
            new("in stream %p, %p is a %s, not a %s" %
                [path, key, items[key].class.human_name, klass.human_name])
        else
          item
        end
      end
    end

    class CollectionRoot < CollectionStream
      def initialize
        super("")
        setup
      end

      def delete(name, opts={})
        super
        # If that didn't fail, it deleted the . stream.  Recreate it.
        setup
      end

      def check_can_merge(src_item, key)
        if items.include?(key)
          super
        else
          raise_root_write_error(key)
        end
      end

      private

      def setup
        items["."] = CollectionStream.new(".")
      end

      def raise_root_write_error(key)
        raise ArgumentError.new("can't write to %p at collection root" % key)
      end

      def []=(key, item)
        raise_root_write_error(key)
      end
    end

    class StreamManifest
      # Build a manifest text for a single stream, without substreams.
      # The manifest includes files in the order they're added.  If you want
      # a normalized manifest, add files in lexical order by name.

      def initialize(name)
        @name = name
        @loc_ranges = {}
        @loc_range_start = 0
        @file_specs = []
      end

      def add_file(coll_file)
        coll_file.each_segment do |segment|
          extend_locator_ranges(segment.locators)
          extend_file_specs(coll_file.name, segment)
        end
      end

      def to_s
        if @file_specs.empty?
          ""
        else
          "%s %s %s\n" % [escape_name(@name),
                          @loc_ranges.keys.join(" "),
                          @file_specs.join(" ")]
        end
      end

      private

      def extend_locator_ranges(locators)
        locators.
            select { |loc_s| not @loc_ranges.include?(loc_s) }.
            each do |loc_s|
          @loc_ranges[loc_s] = LocatorRange.new(loc_s, @loc_range_start)
          @loc_range_start = @loc_ranges[loc_s].end
        end
      end

      def extend_file_specs(filename, segment)
        # Given a filename and a LocatorSegment, add the smallest
        # possible array of file spec strings to @file_specs that
        # builds the file from available locators.
        filename = escape_name(filename)
        start_pos = segment.start_pos
        length = segment.length
        start_loc = segment.locators.first
        prev_loc = start_loc
        # Build a list of file specs by iterating through the segment's
        # locators and preparing a file spec for each contiguous range.
        segment.locators[1..-1].each do |loc_s|
          range = @loc_ranges[loc_s]
          if range.begin != @loc_ranges[prev_loc].end
            range_start, range_length =
              start_and_length_at(start_loc, prev_loc, start_pos, length)
            @file_specs << "#{range_start}:#{range_length}:#{filename}"
            start_pos = 0
            length -= range_length
            start_loc = loc_s
          end
          prev_loc = loc_s
        end
        range_start, range_length =
          start_and_length_at(start_loc, prev_loc, start_pos, length)
        @file_specs << "#{range_start}:#{range_length}:#{filename}"
      end

      def escape_name(name)
        name.gsub(/\\/, "\\\\\\\\").gsub(/\s/) do |s|
          s.each_byte.map { |c| "\\%03o" % c }.join("")
        end
      end

      def start_and_length_at(start_key, end_key, start_pos, length)
        range_begin = @loc_ranges[start_key].begin + start_pos
        range_length = [@loc_ranges[end_key].end - range_begin, length].min
        [range_begin, range_length]
      end
    end
  end
end
