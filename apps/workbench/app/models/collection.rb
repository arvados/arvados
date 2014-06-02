class Collection < ArvadosBase
  MD5_EMPTY = 'd41d8cd98f00b204e9800998ecf8427e'

  # Return true if the given string is the locator of a zero-length blob
  def self.is_empty_blob_locator? locator
    !!locator.to_s.match("^#{MD5_EMPTY}(\\+.*)?\$")
  end

  def self.goes_in_folders?
    true
  end

  def content_summary
    ApplicationController.helpers.human_readable_bytes_html(total_bytes) + " " + super
  end

  def total_bytes
    if files
      tot = 0
      files.each do |file|
        tot += file[2]
      end
      tot
    end
  end

  def files_tree
    return [] if files.empty?
    tree = files.group_by { |file_spec| File.split(file_spec.first) }
    # Fill in entries for empty directories.
    tree.keys.map { |basedir, _| File.split(basedir) }.each do |splitdir|
      until tree.include?(splitdir)
        tree[splitdir] = []
        splitdir = File.split(splitdir.first)
      end
    end
    dir_to_tree = lambda do |dirname|
      # First list subdirectories, with their files inside.
      subnodes = tree.keys.select { |bd, td| (bd == dirname) and (td != '.') }
        .sort.flat_map do |parts|
        [parts + [nil]] + dir_to_tree.call(File.join(parts))
      end
      # Then extend that list with files in this directory.
      subnodes + tree[File.split(dirname)]
    end
    dir_to_tree.call('.')
  end

  def attribute_editable? attr, *args
    false
  end

  def self.creatable?
    false
  end

  def provenance
    arvados_api_client.api "collections/#{self.uuid}/", "provenance"
  end

  def used_by
    arvados_api_client.api "collections/#{self.uuid}/", "used_by"
  end

end
