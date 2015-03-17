# Be sure to restart your server when you modify this file.

# Add new mime types for use in respond_to blocks:
# Mime::Type.register "text/richtext", :rtf
# Mime::Type.register_alias "text/html", :iphone

# add new mime types to register
Mime::Type.register "application/fa", :fa
Mime::Type.register "application/fasta", :fasta
Mime::Type.register "application/go", :go
Mime::Type.register "application/r", :r
Mime::Type.register "application/sam", :sam

# register MIME type with MIME::Type gem 
require 'mime/types'
include MIME
MIME::Types.add(MIME::Type.from_array("application/fa", %(fa)))
MIME::Types.add(MIME::Type.from_array("application/fasta", %(fasta)))
MIME::Types.add(MIME::Type.from_array("application/go", %(go)))
MIME::Types.add(MIME::Type.from_array("application/r", %(r)))
MIME::Types.add(MIME::Type.from_array("application/sam", %(sam)))
