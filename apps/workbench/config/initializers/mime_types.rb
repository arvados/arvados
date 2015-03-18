# Be sure to restart your server when you modify this file.

# Add new mime types for use in respond_to blocks:
# Mime::Type.register "text/richtext", :rtf
# Mime::Type.register_alias "text/html", :iphone

# add new mime types to register

require 'mime/types'
include MIME

# register and add new MIME types to MIME::Types gem
if (MIME::Types.type_for('file.fa').first.nil?)
  Mime::Type.register "application/fa", :fa
  MIME::Types.add(MIME::Type.new(["application/fa", %(fa)]))
end

if (MIME::Types.type_for('file.fasta').first.nil?)
  Mime::Type.register "application/fasta", :fasta
  MIME::Types.add(MIME::Type.new(["application/fasta", %(fasta)]))
end

if (MIME::Types.type_for('file.go').first.nil?)
  Mime::Type.register "application/go", :go
  MIME::Types.add(MIME::Type.new(["application/go", %(go)]))
end

if (MIME::Types.type_for('file.r').first.nil?)
  Mime::Type.register "application/r", :r
  MIME::Types.add(MIME::Type.new(["application/r", %(r)]))
end

if (MIME::Types.type_for('file.sam').first.nil?)
  Mime::Type.register "application/sam", :sam
  MIME::Types.add(MIME::Type.new(["application/sam", %(sam)]))
end
