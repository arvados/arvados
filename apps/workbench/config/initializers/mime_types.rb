# Be sure to restart your server when you modify this file.

# Add new mime types for use in respond_to blocks:
# Mime::Type.register "text/richtext", :rtf
# Mime::Type.register_alias "text/html", :iphone

# add new mime types to MIME from mime_types gem

require 'mime/types'
include MIME
[
  %w(fasta fa fas fsa seq),
  %w(go),
  %w(r),
  %w(sam),
].each do |suffixes|
  if (MIME::Types.type_for(suffixes[0]).first.nil?)
    MIME::Types.add(MIME::Type.new(["application/#{suffixes[0]}", suffixes]))
  end
end
