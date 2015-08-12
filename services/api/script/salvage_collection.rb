#!/usr/bin/env ruby

# Take two input parameters: a collection uuid and reason
# Get "src" collection with the given uuid
# Create a new collection "dest" with:
#   src.manifest_text as "invalid manifest_text.txt"
#   Locators from src.manifest_text as "salvaged_data"
# Update src collection:
#   Set src.manifest_text to: ""
#   Append to src.name: " (reason; salvaged data at dest.uuid)"
#   Set portable_data_hash to "d41d8cd98f00b204e9800998ecf8427e+0"

require 'trollop'

opts = Trollop::options do
  banner ''
  banner "Usage: salvage_collection.rb " +
    "{uuid} {reason}"
  banner ''
  opt :uuid, <<-eos
uuid of the collection to be salvaged.
  eos
  opt :reason, <<-eos
Reason for salvaging.
  eos
end

if ARGV.count < 1
  Trollop::die "required uuid argument is missing"
end

uuid, reason = ARGV

require File.dirname(__FILE__) + '/../config/environment'
require 'arvados/keep'
include ApplicationHelper
require 'tempfile'

def salvage_collection uuid, reason
  act_as_system_user do
    src = Collection.find_by_uuid uuid
    if !src
      puts "No collection found for #{uuid}"
      return
    end

    begin
      src_manifest_text = src.manifest_text || ''

      # Get all the locators from the original manifest
      locators = []
      src_manifest_text.each_line do |line|
        line.split(' ').each do |word|
          if match = Keep::Locator::LOCATOR_REGEXP.match(word)
            word = word.split('+')[0..1].join('+')  # get rid of any hints
            locators << word
          end
        end
      end
      locators << 'd41d8cd98f00b204e9800998ecf8427e+0' if !locators.any?

      # create new collection using 'arv-put' with original manifest_text as the data
      temp_file = Tempfile.new('temp')
      temp_file.write(src.manifest_text)

      created = %x(arv-put --use-filename invalid_manifest_text.txt #{temp_file.path})

      temp_file.close
      temp_file.unlink

      created.rstrip!
      match = created.match HasUuid::UUID_REGEX
      raise "uuid not found" if !match

      # update this new collection manifest to reference all locators from the original manifest
      new_collection = Collection.find_by_uuid created

      new_manifest = new_collection['manifest_text']
      new_manifest = new_manifest.gsub(/\+A[^+]*/, '')
      total_size = 0
      locators.each do |locator|
        total_size += locator.split('+')[1].to_i
      end
      new_manifest += (". #{locators.join(' ')} 0:#{total_size}:salvaged_data\n")

      new_collection.name = "salvaged from #{src.uuid}, #{src.portable_data_hash}"
      new_collection.manifest_text = new_manifest
      new_collection.portable_data_hash = Digest::MD5.hexdigest(new_manifest)

      new_collection.save!

      puts "Created collection for salvaged #{uuid} with uuid: #{created}  #{match}"
    rescue => error
      puts "Error creating collection for #{uuid}: #{error}"
      return
    end

    begin
      # update src collection name, pdh, and manifest_text
      src.name = (src.name || '') + ' (' + (reason || '') + '; salvaged data at ' + created + ')'
      src.manifest_text = ''
      src.portable_data_hash = 'd41d8cd98f00b204e9800998ecf8427e+0'
      src.save!
      puts "Salvaged collection #{uuid}"
    rescue => error
      puts "Error salvaging collection #{uuid}: #{error}"
    end
  end
end

# Salvage the collection with the given uuid
salvage_collection uuid, reason
