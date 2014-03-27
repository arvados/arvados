# Load the rails application
require File.expand_path('../application', __FILE__)
require 'josh_id'

# Initialize the rails application
Server::Application.initialize!
begin
  Rails.cache.clear
rescue Errno::ENOENT => e
  # Cache directory does not exist? Then cache is clear, proceed.
  Rails.logger.warn "In Rails.cache.clear, ignoring #{e.inspect}"
end
