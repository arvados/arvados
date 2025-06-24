ENV["BUNDLE_GEMFILE"] ||= File.expand_path("../Gemfile", __dir__)

# Setting an environment variable before loading rack is the only way
# to change rack's request size limit for an urlencoded POST body.
# Rack::QueryParser accepts an initialization argument to override the
# default, but rack only ever uses its global default_parser, and
# there is no facility for overriding that at runtime.
#
# Our strategy is to rely on the more configurable downstream servers
# (Nginx and arvados-controller) to reject oversized requests before
# they hit this server at all.
ENV["RACK_QUERY_PARSER_BYTESIZE_LIMIT"] = (4 << 30).to_s

require "bundler/setup" # Set up gems listed in the Gemfile.
