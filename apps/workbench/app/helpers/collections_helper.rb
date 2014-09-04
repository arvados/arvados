module CollectionsHelper
  def d3ify_links(links)
    links.collect do |x|
      {source: x.tail_uuid, target: x.head_uuid, type: x.name}
    end
  end

  ##
  # Regex match for collection UUIDs, returns a regex match object with the
  # hash in group 1, (optional) size in group 2, (optional) subsequent uuid
  # fields in group 3, and (optional) file path within the collection as group
  # 4; or nil for no match.
  #
  # +uuid+ the uuid string to match
  #
  def self.match(uuid)
    /^([a-f0-9]{32})(\+[0-9]+)?(\+.*?)?(\/.*)?$/.match(uuid.to_s)
  end

  ##
  # Regex match for collection UUID or "UUID/file" path.
  # Returns a regex match object with the
  # UUID in group 1, (optional) file path in group 2; or nil for no match.
  #
  def self.match_uuid_with_optional_filepath(uuid_with_optional_file)
    /^([0-9a-z]{5}-[0-9a-z]{5}-[0-9a-z]{15})(\/*.*)$/.match(uuid_with_optional_file.to_s)
  end

  ##
  # Regex match for common image file extensions, returns a regex match object
  # with the matched extension in group 1; or nil for no match.
  #
  # +file+ the file string to match
  #
  def self.is_image file
    /\.(jpg|jpeg|gif|png|svg)$/i.match(file)
  end

  ##
  # Generates a relative file path than can be appended to the URL of a
  # collection to get a file download link without adding a spurious ./ at the
  # beginning for files in the default stream.
  #
  # +file+ an entry in the Collection.files list in the form [stream, name, size]
  #
  def self.file_path file
    f0 = file[0]
    f0 = '' if f0 == '.'
    f0 = f0[2..-1] if f0[0..1] == './'
    f0 += '/' if not f0.empty?
    file_path = "#{f0}#{file[1]}"
  end
end
