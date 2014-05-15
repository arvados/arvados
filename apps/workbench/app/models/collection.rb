class Collection < ArvadosBase
  include ApplicationHelper

  MD5_EMPTY = 'd41d8cd98f00b204e9800998ecf8427e'

  # Return true if the given string is the locator of a zero-length blob
  def self.is_empty_blob_locator? locator
    !!locator.to_s.match("^#{MD5_EMPTY}(\\+.*)?\$")
  end

  def content_summary
    human_readable_bytes_html(total_bytes) + " " + super
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

  def attribute_editable?(attr)
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
