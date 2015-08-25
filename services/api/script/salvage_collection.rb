#!/usr/bin/env ruby

# Take two input parameters: a collection uuid and reason
# Get "src_collection" with the given uuid
# Create a new collection with:
#   src_collection.manifest_text as "invalid_manifest_text.txt"
#   Locators from src_collection.manifest_text as "salvaged_data"
# Update src_collection:
#   Set src_collection.manifest_text to: ""
#   Append to src_collection.name: " (reason; salvaged data at new_collection.uuid)"
#   Set portable_data_hash to "d41d8cd98f00b204e9800998ecf8427e+0"

require 'trollop'
require './lib/salvage_collection'

opts = Trollop::options do
  banner ''
  banner "Usage: salvage_collection.rb " +
    "{uuid} {reason}"
  banner ''
  opt :uuid, "uuid of the collection to be salvaged.", type: :string, required: true
  opt :reason, "Reason for salvaging.", type: :string, required: false
end

# Salvage the collection with the given uuid
SalvageCollection.salvage_collection opts.uuid, opts.reason
