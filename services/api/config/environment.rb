# Load the rails application
require File.expand_path('../application', __FILE__)
require 'josh_id'

# Initialize the rails application
Server::Application.initialize!
Rails.cache.clear
